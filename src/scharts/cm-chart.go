package scharts

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

var (
	reg          = `(\d{4}-\d{1,2}-\d{1,2})_(\d{2}:\d{2}:\d{2})[\s\S]*?S\s+((\d+\.*\d+)\s+(\d+\.*\d+)\s+.+?`
	timeTemplate = "2006-01-02 15:04:05"
)

type node struct {
	XAxis string
	opts.BarData
}

type lineInfo struct {
	lable string
	nodes []*node
}

// func init() {
// 	reg = `(\d{4}-\d{1,2}-\d{1,2})_(\d{2}:\d{2}:\d{2})[\s\S]*?S\s+((\d+\.*\d+)\s+(\d+\.*\d+)\s+.+?`

// }

func readFileMethod(fileName string) string {
	f, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("读取文件失败:%#v", err)
		return ""
	}
	return string(f)
}

func getFileAxis(path, dataType string, reg *regexp.Regexp) []*node {
	fileString := readFileMethod(path)
	items := reg.FindAllStringSubmatch(fileString, -1)

	nodes := make([]*node, 0)
	for _, item := range items {
		cpu, _ := strconv.ParseFloat(item[4], 32)
		men, _ := strconv.ParseFloat(item[5], 32)

		xNode := &node{
			XAxis: item[2],
		}
		switch dataType {
		case "CPU":
			xNode.Value = float32(cpu)
		case "MEM":
			xNode.Value = float32(men)
		default:
			xNode.Value = float32(cpu)
		}

		nodes = append(nodes, xNode)
	}
	return nodes
}

func getAxis(path, fileStart, dataType string, reg *regexp.Regexp) []*node {
	nodeSlice := make([]*node, 0)
	files, _ := ioutil.ReadDir(path)
	for _, file := range files {

		filePath := path + "\\" + file.Name()
		if file.IsDir() {
			nodeSlice = append(nodeSlice, getAxis(filePath, fileStart, dataType, reg)...)
		} else if strings.HasPrefix(file.Name(), fileStart) {
			nodeSlice = append(nodeSlice, getFileAxis(filePath, dataType, reg)...)
		}
	}
	return nodeSlice
}

func newLine(server, instance, dataType string, lineInfo *lineInfo) *charts.Bar {
	xAxis := make([]string, len(lineInfo.nodes))
	for index, node := range lineInfo.nodes {
		xAxis[index] = node.XAxis
	}

	line := charts.NewBar()
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1800px",
			Height: "400px",
		}),
		//charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithTitleOpts(opts.Title{
			Title: server + " " + instance + " " + lineInfo.lable + " " + dataType,
			//Subtitle: instance + " " + lineInfo.lable,
		}),
		charts.WithColorsOpts(opts.Colors{"black"}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: dataType + "%",
			SplitLine: &opts.SplitLine{
				Show: false,
			},
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "time",
		}),
	)
	line = line.SetXAxis(xAxis)

	lineData := make([]opts.BarData, len(lineInfo.nodes))
	for i, node := range lineInfo.nodes {
		lineData[i] = node.BarData
	}

	line = line.AddSeries(lineInfo.lable, lineData)

	//AddSeries("men", generateLineItems()).
	line.SetSeriesOptions(
		// charts.WithLabelOpts(opts.Label{
		// 	Show: true,
		// }),
		// charts.WithMarkPointStyleOpts(opts.MarkPointStyle{
		// 	Label: &opts.Label{
		// 		Show:      true,
		// 		Formatter: "{a}: {b}",
		// 	},
		// }),
		// charts.WithAreaStyleOpts(opts.AreaStyle{
		// 	Opacity: 0.2,
		// }),
		// charts.WithLineChartOpts(opts.LineChart{
		// 	Smooth: true,
		// }),
		//charts.WithLineStyleOpts(opts.LineStyle{Type: "dotted"}),
		// charts.WithMarkPointNameTypeItemOpts(
		// 	opts.MarkPointNameTypeItem{Name: "Maximum", Type: "max"},
		// 	opts.MarkPointNameTypeItem{Name: "Average", Type: "average"},
		// 	opts.MarkPointNameTypeItem{Name: "Minimum", Type: "min"},
		// ),
		charts.WithMarkLineNameTypeItemOpts(
			opts.MarkLineNameTypeItem{Name: "Maximum", Type: "max"},
			opts.MarkLineNameTypeItem{Name: "Average", Type: "average"},
			opts.MarkLineNameTypeItem{Name: "Minimum", Type: "min"},
		),
	// charts.WithMarkPointStyleOpts(
	// 	opts.MarkPointStyle{Label: &opts.Label{Show: true}},
	// ),
	)

	return line
}

func CMChart(w http.ResponseWriter, request *http.Request) {
	querys := request.URL.Query()
	dataType := strings.ToUpper(querys.Get("type"))
	server := querys.Get("server")
	instance := querys.Get("instance")
	logdate := querys.Get("date")

	if dataType != "CPU" && dataType != "MEM" {
		dataType = "CPU"
	}

	// Put data into instance
	reg := regexp.MustCompile(reg + instance + `)`)

	dateArr := strings.Split(logdate, ",")
	dateArrLen := len(dateArr)
	syncLock := new(sync.WaitGroup)
	syncLock.Add(dateArrLen)

	lineInfos := make([]*lineInfo, dateArrLen)

	for index, date := range dateArr {
		go func(i int, d string) {
			defer syncLock.Done()

			lineInfos[i] = &lineInfo{
				lable: d,
				nodes: getAxis(`.\logs\`+server, d, dataType, reg),
			}

		}(index, date)
	}
	syncLock.Wait()

	for _, lineInfo := range lineInfos {
		// create a new line instance
		line := newLine(server, instance, dataType, lineInfo)
		line.Render(w)
	}
	fmt.Println(time.Now().Format(timeTemplate), "生成图表：", server, dataType, instance, logdate)
}
