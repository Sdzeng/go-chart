package scharts

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

var (
	reg string
)

type node struct {
	date string
	cpu  float32
	men  float32
}

func init() {
	reg = `(\d{4}-\d{1,2}-\d{1,2}_\d{2}:\d{2}:\d{2})[\s\S]*?S\s+((\d+\.*\d+)\s+(\d+\.*\d+)\s+.+?`

}

// func generateLineItems() []opts.LineData {
// 	items := make([]opts.LineData, 0)
// 	for i := 0; i < 7; i++ {
// 		items = append(items, opts.LineData{Value: rand.Intn(300)})
// 	}
// 	return items
// }

func readFileMethod(fileName string) string {
	f, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("读取文件失败:%#v", err)
		return ""
	}
	return string(f)
}

func getFileAxis(path string, reg *regexp.Regexp) []*node {
	fileString := readFileMethod(path)
	items := reg.FindAllStringSubmatch(fileString, -1)

	nodes := make([]*node, 0)
	for _, item := range items {
		cpu, _ := strconv.ParseFloat(item[3], 32)
		men, _ := strconv.ParseFloat(item[4], 32)
		xNode := &node{
			date: item[1],
			cpu:  float32(cpu),
			men:  float32(men),
		}

		nodes = append(nodes, xNode)

	}
	return nodes
}

func getAxis(path, fileStart string, reg *regexp.Regexp) []*node {
	nodes := make([]*node, 0)
	//path:=`D:\GitRepository\go-chart\src\logs\226_20210823-25`
	files, _ := ioutil.ReadDir(path)
	for _, file := range files {

		filePath := path + "\\" + file.Name()
		if file.IsDir() {
			nodes = append(nodes, getAxis(filePath, fileStart, reg)...)
		} else if strings.HasPrefix(file.Name(), fileStart) {
			nodes = append(nodes, getFileAxis(filePath, reg)...)
		}
	}
	return nodes
}

func CMChart(w http.ResponseWriter, request *http.Request) {
	querys := request.URL.Query()
	dateType := querys.Get("type")
	server := querys.Get("server")
	instance := querys.Get("instance")
	logdate := querys.Get("date")

	if dateType == "" {
		dateType = "CPU"
	} else {
		dateType = strings.ToUpper(dateType)
	}

	// create a new line instance
	line := charts.NewBar()
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1800px",
			Height: "600px",
		}),
		//charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title:    server + " " + dateType,
			Subtitle: instance + " " + logdate,
		}),
		charts.WithColorsOpts(opts.Colors{"black"}),
	)

	// Put data into instance
	reg := regexp.MustCompile(reg + instance + `)`)
	nodes := getAxis(`.\logs\`+server, logdate, reg)
	xAxis := make([]string, len(nodes))
	lineDatas := make([]opts.BarData, len(nodes))
	for index, xNode := range nodes {
		xAxis[index] = xNode.date
		switch dateType {
		case "CPU":
			lineDatas[index] = opts.BarData{Value: xNode.cpu}
		case "MEM":
			lineDatas[index] = opts.BarData{Value: xNode.men}
		default:
			lineDatas[index] = opts.BarData{Value: xNode.cpu}
		}

	}

	line.SetXAxis(xAxis).
		//AddSeries("cpu", cpuLine).
		AddSeries(dateType, lineDatas).
		//AddSeries("men", generateLineItems()).
		SetSeriesOptions(
			// charts.WithLabelOpts(opts.Label{
			// 	Show: true,
			// }),
			// charts.WithAreaStyleOpts(opts.AreaStyle{
			// 	Opacity: 0.5,
			// }),

			charts.WithMarkPointNameTypeItemOpts(
				opts.MarkPointNameTypeItem{Name: "Maximum", Type: "max"},
				opts.MarkPointNameTypeItem{Name: "Average", Type: "average"},
				opts.MarkPointNameTypeItem{Name: "Minimum", Type: "min"},
			),
		// charts.WithMarkPointStyleOpts(
		// 	opts.MarkPointStyle{Label: &opts.Label{Show: true}},
		// ),
		)
	line.Render(w)
	fmt.Println("生成图表：", server, dateType, instance, logdate)
}
