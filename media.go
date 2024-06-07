package main

import (
	"errors"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"regexp"
)

type MediaMeta struct {
	TitleId string
	Name    string

	Type string
}

type Media struct {
	MediaMeta

	Thumbnail Image
	Images    []Image
}

const (
	NameXpath      = "//a/h3"
	IdXpath        = "//div/span[1]"
	TypeXpath      = "//div/span[2]"
	ThumbnailXpath = "//img[contains(@alt, \"Title picture\")]/@data-src"
	PicXpath       = "//div[contains(@class, \"gamerpic\")]/div/img/@data-src"
)

var (
	ErrNameGet      = errors.New("error getting title name")
	ErrIdGet        = errors.New("error getting title id")
	ErrTypeGet      = errors.New("error getting title type")
	ErrThumbnailGet = errors.New("error getting thumbnail")
	ErrPicGet       = errors.New("error getting gamerpic")
)

func getMediaFromDiv(div *html.Node) (*Media, error) {
	var err error
	var m Media
	var n *html.Node
	var nn []*html.Node

	//name
	n, err = htmlquery.Query(div, NameXpath)
	if err != nil || n == nil {
		return nil, errors.Join(ErrNameGet, err)
	}
	m.Name = n.FirstChild.Data
	//id
	n, err = htmlquery.Query(div, IdXpath)
	if err != nil || n == nil {
		return nil, errors.Join(ErrIdGet, err)
	}
	m.TitleId = n.FirstChild.Data
	//type
	n, err = htmlquery.Query(div, TypeXpath)
	if err != nil || n == nil {
		return nil, errors.Join(ErrTypeGet, err)
	}
	m.Type = n.FirstChild.Data

	//thumbnail
	n, err = htmlquery.Query(div, ThumbnailXpath)
	if err != nil || n == nil {
		return nil, errors.Join(ErrThumbnailGet, err)
	}
	m.Thumbnail.Url = n.FirstChild.Data
	m.Thumbnail.TitleId, m.Thumbnail.Filename, err = urlToTitleName(m.Thumbnail.Url)
	if err != nil {
		return nil, err
	}

	//gamer pics
	nn, err = htmlquery.QueryAll(div, PicXpath)
	if err != nil || nn == nil {
		return nil, errors.Join(ErrPicGet, err)
	}
	m.Images = make([]Image, len(nn))
	for i := 0; i < len(nn); i++ {
		m.Images[i].Url = nn[i].FirstChild.Data
		m.Images[i].TitleId, m.Images[i].Filename, err = urlToTitleName(m.Images[i].Url)
		if err != nil {
			return nil, err
		}
	}

	return &m, err
}

var nameRegex = regexp.MustCompile("titles/[a-f0-9]{8}/.+")
var ErrUrlParse = errors.New("error parsing url for title and filename")

func urlToTitleName(url string) (string, string, error) {
	url = nameRegex.FindString(url)

	if len(url) < 16 {
		return "", "", ErrUrlParse
	}

	return url[7:15], url[16:], nil
}

type Image struct {
	TitleId  string
	Filename string

	Url string
}
