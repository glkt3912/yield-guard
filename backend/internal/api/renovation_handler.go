package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yield-guard/backend/internal/domain"
)

func validateRenovationInput(in domain.RenovationInput) error {
	if in.PropertyPrice <= 0 {
		return errors.New("propertyPrice は正の値を指定してください")
	}
	if in.AnnualBaseRent < 0 {
		return errors.New("annualBaseRent は 0 以上を指定してください")
	}
	if in.EffectiveTaxRate < 0 || in.EffectiveTaxRate > 1 {
		return errors.New("effectiveTaxRate は 0.0〜1.0 の範囲で指定してください")
	}
	if in.SelfLaborRatePerHour < 0 {
		return errors.New("selfLaborRatePerHour は 0 以上を指定してください")
	}
	if len(in.Items) == 0 {
		return errors.New("items は 1 件以上指定してください")
	}
	for idx, item := range in.Items {
		if item.Cost <= 0 {
			return fmt.Errorf("items[%d].cost は正の値を指定してください", idx)
		}
	}
	return nil
}

// HandleRenovationAnalyze はリフォームROIシミュレーションを実行する
// @Summary     リフォームROI分析
// @Tags        renovation
// @Accept      json
// @Produce     json
// @Param       body  body  domain.RenovationInput  true  "リフォーム分析リクエスト"
// @Success     200  {object}  domain.RenovationResult
// @Failure     400  {object}  map[string]string
// @Router      /api/renovation/analyze [post]
func (h *Handler) HandleRenovationAnalyze(c *gin.Context) {
	var input domain.RenovationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		badRequest(c, "リクエストの形式が不正です: "+err.Error())
		return
	}
	if err := validateRenovationInput(input); err != nil {
		badRequest(c, err.Error())
		return
	}
	result := domain.CalcRenovationROI(input)
	c.JSON(http.StatusOK, result)
}
