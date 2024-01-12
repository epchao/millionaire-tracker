package handlers

import (
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
	line.PageTitle = "Millionaire Tracker"
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Millionaire Tracker @thejosephmurray",
			Subtitle: "How long will it take for them to become a millionaire based on their pure income and expenditures?",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: true, Trigger: "axis"}),
		charts.WithToolboxOpts(opts.Toolbox{Show: true}),
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
			charts.WithLineChartOpts(opts.LineChart{Smooth: true}),
		)

	c.Type("html")
	line.Render(c.Context())

	return nil
}
