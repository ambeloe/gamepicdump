package main

import (
	"flag"
	"fmt"
	"github.com/antchfx/htmlquery"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/html"
	"io"
	"math"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
)

var nameRegex = regexp.MustCompile("titles/[a-f0-9]{8}/.+")

func main() {
	os.Exit(rMain())
}

func rMain() int {
	var err error

	var buf []byte
	var currPage = "https://xboxgamer.pics/titles/all?page="
	var page *html.Node
	var pageNum = 1
	var maxPage = math.MaxInt - 1
	var prog *progressbar.ProgressBar
	var res []*html.Node
	var resp *http.Response

	var outDir = flag.String("o", "", "output directory")

	flag.Parse()

	for pageNum < maxPage+1 {
		page, err = htmlquery.LoadURL(currPage + strconv.Itoa(pageNum))
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error downloading and parsing page:", err)
			return 1
		}

		if maxPage == math.MaxInt-1 {
			res, err = htmlquery.QueryAll(page, "//*[@id=\"content\"]/section/div/a[3]")
			if err != nil || len(res) == 0 {
				_, _ = fmt.Fprintln(os.Stderr, "error getting last page:", err)
				return 1
			}

			maxPage, err = strconv.Atoi(res[0].FirstChild.Data)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "failed to parse page number:", err)
				return 1
			}
			fmt.Println(strconv.Itoa(maxPage), "pages found")
		}

		//this doesn't work in this xpath parser, but works in chrome
		//div[@id="content"]/section/div/div/div/div/a[contains(@href,"download.xboxgamer.pics/titles/")]/@href
		res, err = htmlquery.QueryAll(page, "//a[contains(@href,\"download.xboxgamer.pics/titles/\")]/@href")
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error searching for elements in page:", err)
			return 1
		}

		if len(res) == 0 {
			_, _ = fmt.Fprintln(os.Stderr, "no gamer pictures found on page, quitting")
			return 1
		} else {
			prog = progressbar.Default(int64(len(res)), fmt.Sprintf("Page %d", pageNum))
			for i := 0; i < len(res); i++ {
				name := nameRegex.FindString(res[i].FirstChild.Data)

				_, err = os.Open(path.Join(*outDir, name[7:15], name[16:]))
				if err == nil {
					continue
				}

				resp, err = http.Get(res[i].FirstChild.Data)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "failed to get pic %s: %v", res[i].FirstChild.Data, err)
					return 1
				}
				buf, err = io.ReadAll(resp.Body)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "failed to download pic %s: %v", res[i].FirstChild.Data, err)
					return 1
				}

				err = os.MkdirAll(path.Join(*outDir, name[7:15]), 0755)
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "could not create directory:", err)
					return 1
				}
				err = os.WriteFile(path.Join(*outDir, name[7:15], name[16:]), buf, 0644)
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "failed to write image:", err)
					return 1
				}

				_ = prog.Add(1)
			}
			_ = prog.Finish()
			pageNum++
		}
	}

	return 0
}
