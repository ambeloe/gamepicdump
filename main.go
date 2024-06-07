package main

import (
	"encoding/json"
	"errors"
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
	"strconv"
	"sync/atomic"
)

func main() {
	os.Exit(rMain())
}

func rMain() int {
	var err error

	var buf []byte
	var currPage = "https://xboxgamer.pics/titles/all?page="
	var page *html.Node
	var pageNum = 1
	var m *Media
	var maxPage = math.MaxInt - 1
	var res []*html.Node

	var numImgs atomic.Int32
	var imgs = make(chan Image, 1e6)
	var errs = make(chan error, 1e6)
	var prog *progressbar.ProgressBar

	var outDir = flag.String("o", "", "output directory")
	var threads = flag.Uint("t", 16, "number of threads to use for concurrent downloads")

	flag.Parse()

	for i := 0; i < int(*threads); i++ {
		go downloadWorker(i, *outDir, imgs, errs)
	}

	fmt.Println("scraping pages...")
	prog = progressbar.Default(int64(maxPage), "")
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
			//fmt.Println(strconv.Itoa(maxPage), "pages found")
			prog.ChangeMax(maxPage)
		}

		//this doesn't work in this xpath parser, but works in chrome
		//div[@id="content"]/section/div/div/div/div/a[contains(@href,"download.xboxgamer.pics/titles/")]/@href
		//overcomplicated; kept for posterity
		//div[@id="content"]/section/div[//img[contains(@data-src,"assets.xboxgamer.pics/titles/")] and contains(@class, "title")]
		res, err = htmlquery.QueryAll(page, "//div[contains(@class, \"title\")]")
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "error searching for elements in page:", err)
			return 1
		}

		if len(res) == 0 {
			_, _ = fmt.Fprintln(os.Stderr, "no gamer pictures found on page, quitting")
			return 1
		} else {
			for i := 0; i < len(res); i++ {
				m, err = getMediaFromDiv(res[i])
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "error parsing media entry:", err)
					return 1
				}

				err = os.MkdirAll(path.Join(*outDir, m.TitleId), 0755)
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "failed to create output folder:", err)
					return 1
				}

				buf, err = json.MarshalIndent(m.MediaMeta, "", "\t")
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "failed to marshal metadata:", err)
					return 1
				}
				err = os.WriteFile(path.Join(*outDir, m.TitleId, "metadata.json"), buf, 0644)
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "failed to write metadata:", err)
					return 1
				}

				//queue images for download
				numImgs.Add(1)
				imgs <- m.Thumbnail
				for j := 0; j < len(m.Images); j++ {
					numImgs.Add(1)
					imgs <- m.Images[j]
				}
			}
		}
		pageNum++
		_ = prog.Add(1)
	}
	_ = prog.Finish()

	fmt.Println("waiting on image downloads...")
	prog = progressbar.Default(int64(numImgs.Load()), "")
	for i := 0; i < int(numImgs.Load()); i++ {
		err = <-errs
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Worker error: %v\n", err)
			return 1
		}
		_ = prog.Add(1)
	}
	_ = prog.Finish()

	return 0
}

var ErrDownloadRequest = errors.New("failed to make file request")
var ErrDownload = errors.New("failed to download file")
var ErrFileWrite = errors.New("failed to write file")

// directory must be valid before using
func downloadWorker(id int, baseDir string, images <-chan Image, result chan<- error) {
	var err error
	var buf []byte
	var resp *http.Response

	for i := range images {
		resp, err = http.Get(i.Url)
		if err != nil {
			result <- errors.Join(ErrDownloadRequest, err)
			continue
		}
		buf, err = io.ReadAll(resp.Body)
		if err != nil {
			result <- errors.Join(ErrDownload, err)
			continue
		}
		_ = resp.Body.Close()

		err = os.WriteFile(path.Join(baseDir, i.TitleId, i.Filename), buf, 0644)
		if err != nil {
			result <- errors.Join(ErrFileWrite, err)
			continue
		}

		result <- nil
	}
}
