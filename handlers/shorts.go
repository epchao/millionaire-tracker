package handlers

import (
	"math"
	"sort"

	"github.com/epchao/millionaire-tracker/database"
	"github.com/epchao/millionaire-tracker/models"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/gofiber/fiber/v2"
)

func Visualizer(c *fiber.Ctx) error {
	shorts := []models.Short{}
	database.DB.Db.Find(&shorts)

	sort.Slice(shorts, func(i, j int) bool {
		return shorts[i].Title < shorts[j].Title
	})

	line := charts.NewLine()
	line.PageTitle = "Millionaire Tracker - Visualizer"
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Millionaire Tracker @thejosephmurray",
			Link:     "/analysis",
			Target:   "self",
			Subtitle: "Click the title for an analysis report",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithToolboxOpts(opts.Toolbox{Show: opts.Bool(true)}),
	)

	var days []int
	var revenue []opts.LineData
	var expenses []opts.LineData
	var balance []int
	var balancePlot []opts.LineData

	for _, short := range shorts {
		days = append(days, short.Title)
		revenue = append(revenue, opts.LineData{Value: short.Revenue})
		expenses = append(expenses, opts.LineData{Value: short.Expenses})
		balance = append(balance, short.NetResult)
	}

	for i := 1; i < len(balance); i += 1 {
		balance[i] += balance[i-1]
	}

	for i := 0; i < len(balance); i += 1 {
		balancePlot = append(balancePlot, opts.LineData{Value: balance[i]})
	}

	line.SetXAxis(days).
		AddSeries("Revenue", revenue).
		AddSeries("Expenses", expenses).
		AddSeries("Balance", balancePlot).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{Smooth: opts.Bool(true)}),
		)

	c.Type("html")
	line.Render(c.Context())

	return nil
}

func Analysis(c *fiber.Ctx) error {
	shorts := []models.Short{}
	database.DB.Db.Find(&shorts)

	sort.Slice(shorts, func(i, j int) bool {
		return shorts[i].Title < shorts[j].Title
	})

	totalDays := len(shorts)
	var revenue []int
	var totalRevenue int
	var expenses []int
	var totalExpenses int
	var balance []int

	for _, short := range shorts {
		revenue = append(revenue, short.Revenue)
		totalRevenue += short.Revenue
		expenses = append(expenses, short.Expenses)
		totalExpenses += short.Expenses
		balance = append(balance, short.NetResult)
	}

	for i := 1; i < len(balance); i += 1 {
		balance[i] += balance[i-1]
	}

	currentBalance := balance[len(balance)-1]

	totalSavings := currentBalance + totalRevenue - totalExpenses

	dailyNetSavings := (totalRevenue - totalExpenses) / totalDays

	averageRevenue := totalRevenue / totalDays
	averageExpenses := totalExpenses / totalDays

	sumRevenue := float64(0)
	sumExpenses := float64(0)
	for i := 0; i < totalDays; i += 1 {
		sumRevenue += math.Pow(float64(revenue[i]-averageRevenue), 2)
		sumExpenses += math.Pow(float64(expenses[i]-averageExpenses), 2)
	}

	varianceRevenue := sumRevenue / float64(totalDays-1)
	varianceExpenses := sumExpenses / float64(totalDays-1)

	stdRevenue := math.Sqrt(varianceRevenue)
	stdExpenses := math.Sqrt(varianceExpenses)

	return c.Render("analysis", fiber.Map{
		"CurrentBalance":            currentBalance,
		"TotalRevenue":              totalRevenue,
		"TotalExpenses":             totalExpenses,
		"TotalDays":                 totalDays,
		"TotalSavings":              totalSavings,
		"DailyNetSavings":           dailyNetSavings,
		"DaysTillRich":              (1000000 - totalSavings) / dailyNetSavings,
		"AverageRevenue":            averageRevenue,
		"VarianceRevenue":           varianceRevenue,
		"StandardDeviationRevenue":  stdRevenue,
		"AverageExpenses":           averageExpenses,
		"VarianceExpenses":          varianceExpenses,
		"StandardDeviationExpenses": stdExpenses,
	})
}
