package api

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// logSafeParams はアクセスログに記録してよいクエリパラメータキーの許可リスト。
// PII になりうるキー（address 等）はここに列挙しないこと。
// 起動後に書き換えてはならない（read-only として扱う）。
var logSafeParams = map[string]bool{
	"area": true, "city": true, "prefecture": true, "municipality": true,
	"year": true, "quarter": true, "to_year": true, "to_quarter": true,
	"z": true, "x": true, "y": true,
	"lat": true, "lng": true,
	"minLat": true, "maxLat": true, "minLng": true, "maxLng": true,
	"price": true, "area_sqm": true, "building_age": true,
	"station_minutes": true, "ridership_score": true,
	"budget": true, "yield": true,
	"division": true,
}

// sanitizeQuery はクエリ文字列から許可リスト外のパラメータ値を [REDACTED] に置換して返す。
// url.Values.Encode() は値を URL エンコードするため使用せず、ログ可読性を保つため手動でビルドする。
func sanitizeQuery(raw string) string {
	vals, err := url.ParseQuery(raw)
	if err != nil {
		return "[UNPARSEABLE]"
	}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	first := true
	for _, k := range keys {
		for _, v := range vals[k] {
			if !first {
				sb.WriteByte('&')
			}
			first = false
			sb.WriteString(url.QueryEscape(k))
			sb.WriteByte('=')
			if logSafeParams[k] {
				sb.WriteString(url.QueryEscape(v))
			} else {
				sb.WriteString("[REDACTED]")
			}
		}
	}
	return sb.String()
}

// coordsGlobal and coordsJapanOnly are passed as the japanOnly argument of parseLatLng
// to make call sites self-documenting.
const (
	coordsGlobal    = false
	coordsJapanOnly = true
)

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func badGateway(c *gin.Context, msg string) {
	c.JSON(http.StatusBadGateway, gin.H{"error": msg})
}

// parseLatLng parses lat and lng query params with range validation.
// If japanOnly is true, validates Japan domestic range (lat 20-46, lng 122-154).
// Otherwise validates global range (lat -90~90, lng -180~180).
func parseLatLng(c *gin.Context, japanOnly bool) (lat, lng float64, ok bool) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	if latStr == "" || lngStr == "" {
		badRequest(c, "lat と lng は必須パラメータです")
		return 0, 0, false
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if japanOnly {
		if err != nil || lat < 20 || lat > 46 {
			badRequest(c, "lat は日本国内の緯度（20〜46）で指定してください")
			return 0, 0, false
		}
	} else {
		if err != nil || lat < -90 || lat > 90 {
			badRequest(c, "lat は -90〜90 の数値で指定してください")
			return 0, 0, false
		}
	}
	lng, err = strconv.ParseFloat(lngStr, 64)
	if japanOnly {
		if err != nil || lng < 122 || lng > 154 {
			badRequest(c, "lng は日本国内の経度（122〜154）で指定してください")
			return 0, 0, false
		}
	} else {
		if err != nil || lng < -180 || lng > 180 {
			badRequest(c, "lng は -180〜180 の数値で指定してください")
			return 0, 0, false
		}
	}
	return lat, lng, true
}

// parseZoom parses the optional z query param (11-15). Returns defaultZ if absent.
func parseZoom(c *gin.Context, defaultZ int) (z int, ok bool) {
	zStr := c.Query("z")
	if zStr == "" {
		return defaultZ, true
	}
	zv, err := strconv.Atoi(zStr)
	if err != nil || zv < 11 || zv > 15 {
		badRequest(c, "z は 11〜15 の整数で指定してください")
		return 0, false
	}
	return zv, true
}
