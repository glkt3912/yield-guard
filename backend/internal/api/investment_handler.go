package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
	"github.com/yield-guard/backend/internal/telemetry"
)

// Analyze は投資シミュレーションを実行する
// POST /api/analyze
func (h *Handler) Analyze(c *gin.Context) {
	var input domain.InvestmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, "リクエストの形式が不正です: "+err.Error())
		return
	}

	if err := validateInvestmentInput(input); err != nil {
		badRequest(c, err.Error())
		return
	}

	input.Defaults()
	if err := input.Validate(); err != nil {
		badRequest(c, err.Error())
		return
	}

	result := domain.Analyze(c.Request.Context(), input)

	// run Gemini in background; collect result within remaining context budget
	aiCh := make(chan string, 1)
	go func() {
		aiCh <- h.summarizer.GenerateSummary(c.Request.Context(), result)
	}()
	select {
	case result.AISummary = <-aiCh:
	case <-c.Request.Context().Done():
		// request cancelled or timed out upstream
	}

	telemetry.AnalyzeRequestsTotal.Add(c.Request.Context(), 1)
	c.JSON(http.StatusOK, result)
}

// MonteCarlo はモンテカルロシミュレーションを実行する
// POST /api/investment/simulate
func (h *Handler) MonteCarlo(c *gin.Context) {
	var input domain.MonteCarloInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, "リクエストの形式が不正です: "+err.Error())
		return
	}
	if err := validateInvestmentInput(input.Base); err != nil {
		badRequest(c, err.Error())
		return
	}
	input.Base.Defaults()
	if err := input.Base.Validate(); err != nil {
		badRequest(c, err.Error())
		return
	}
	result := domain.MonteCarloSimulate(input)
	c.JSON(http.StatusOK, result)
}

// GetRentDeclineHint は XCT001 直近5年分から賃料下落率参考値を返す
// GET /api/investment/rent-decline-hint?area=13[&municipality=13101]
func (h *Handler) GetRentDeclineHint(c *gin.Context) {
	area := c.Query("area")
	if area == "" {
		badRequest(c, "area は必須パラメータです")
		return
	}

	municipality := c.Query("municipality")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// XCT001 の対応年（2022〜2026）を並列取得
	years := []int{2022, 2023, 2024, 2025, 2026}
	type yearResult struct {
		year  int
		items []domain.LandAppraisalItem
		err   error
	}
	ch := make(chan yearResult, len(years))
	for _, y := range years {
		go func() {
			items, err := h.mlitClient.FetchLandAppraisals(ctx, area, municipality, y, "00")
			ch <- yearResult{year: y, items: items, err: err}
		}()
	}

	itemsByYear := make(map[int][]domain.LandAppraisalItem, len(years))
	var fetchErr error
	for range years {
		r := <-ch
		if r.err != nil {
			slog.WarnContext(ctx, "FetchLandAppraisals failed", "year", r.year, "area", area, "error", r.err)
			fetchErr = r.err
			continue
		}
		if len(r.items) > 0 {
			itemsByYear[r.year] = r.items
		}
	}

	// 全年エラーの場合のみ502を返す
	if len(itemsByYear) == 0 && fetchErr != nil {
		slog.ErrorContext(c.Request.Context(), "FetchLandAppraisals failed for all years", "err", fetchErr)
		badGateway(c, "地価公示APIからのデータ取得に失敗しました")
		return
	}

	hint := domain.CalcRentDeclineHint(itemsByYear)
	c.JSON(http.StatusOK, hint)
}

// validateInvestmentInput は投資入力値の範囲チェックを行う
func validateInvestmentInput(in domain.InvestmentInput) error {
	if in.LandPrice <= 0 || in.LandPrice > 10_000_000_000 {
		return errors.New("landPrice は 1〜100億円の範囲で指定してください")
	}
	if in.BuildingCost <= 0 || in.BuildingCost > 10_000_000_000 {
		return errors.New("buildingCost は 1〜100億円の範囲で指定してください")
	}
	if in.MonthlyRent <= 0 {
		return errors.New("monthlyRent は正の値を指定してください")
	}
	if in.VacancyRate < 0 || in.VacancyRate >= 1.0 {
		return errors.New("vacancyRate は 0.0〜0.99 の範囲で指定してください")
	}
	if in.LoanAmount < 0 {
		return errors.New("loanAmount は 0 以上を指定してください")
	}
	if in.AnnualLoanRate < 0 || in.AnnualLoanRate > 0.3 {
		return errors.New("annualLoanRate は 0〜30% の範囲で指定してください")
	}
	if in.LoanYears < 0 || in.LoanYears > 50 {
		return errors.New("loanYears は 0〜50 年の範囲で指定してください")
	}
	if in.MiscExpenseRate < 0 || in.MiscExpenseRate > 0.5 {
		return errors.New("miscExpenseRate は 0〜50% の範囲で指定してください")
	}
	if in.ExpenseRate < 0 || in.ExpenseRate > 0.9 {
		return errors.New("expenseRate は 0〜90% の範囲で指定してください")
	}
	if in.IncomeTaxRate < 0 || in.IncomeTaxRate > 0.6 {
		return errors.New("incomeTaxRate は 0〜60% の範囲で指定してください")
	}
	if in.ExitYieldTarget <= 0 || in.ExitYieldTarget > 0.5 {
		return errors.New("exitYieldTarget は 0%超〜50% の範囲で指定してください（ゼロ除算防止）")
	}
	if in.HoldingYears < 0 || in.HoldingYears > 50 {
		return errors.New("holdingYears は 0〜50 年の範囲で指定してください")
	}
	if in.RentDeclineRate < 0 || in.RentDeclineRate > 0.2 {
		return errors.New("rentDeclineRate は 0.0〜0.2 の範囲で指定してください")
	}
	// DiscountRate == 0 は「未指定」扱い: Defaults() で 0.05 に補完される
	if in.DiscountRate < 0 || in.DiscountRate > 0.30 {
		return errors.New("discountRate は 0〜30% の範囲で指定してください")
	}
	if in.PriceDeclineRate < 0 || in.PriceDeclineRate > 0.10 {
		return errors.New("priceDeclineRate は 0〜10% の範囲で指定してください")
	}
	if in.DepreciationMethod != "" &&
		in.DepreciationMethod != domain.DepreciationMethodStraightLine &&
		in.DepreciationMethod != domain.DepreciationMethodDecliningBalance {
		return errors.New("depreciationMethod は \"straight-line\" または \"declining-balance\" を指定してください")
	}
	return nil
}
