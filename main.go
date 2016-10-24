package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"encoding/json"

	"os"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

var (
	baseURL = "https://www.milanuncios.com"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

type extractor func(*html.Node) string

type ad struct {
	Age      string
	Price    string
	Year     string
	Kms      string
	Make     string
	Model    string
	Location string
	URL      string
}

func newRequest(queryValues url.Values) (*http.Request, error) {
	u, _ := url.Parse(baseURL)
	u.Path = "motos-de-carretera/abs.htm"
	// u.Path = "motos-de-carretera-en-barcelona/abs.htm"
	u.RawQuery = queryValues.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func newSearchValues(priceMin, priceMax, yearMin, yearMax, ccMin, ccMax, kmMax string) url.Values {
	return url.Values{
		"desde": []string{priceMin},
		"hasta": []string{priceMax},
		"anod":  []string{yearMin},
		"anoh":  []string{yearMax},
		"ccd":   []string{ccMin},
		"cch":   []string{ccMax},
		"kms":   []string{kmMax},
		"cerca": []string{"s"},
	}
}

func main() {
	ads := fetchNewAds()
	b, err := json.MarshalIndent(ads, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	os.Stdout.Write(b)
}

func fetchNewAds() []ad {

	searchValues := newSearchValues(
		"1000", "4000", "2010", "2016", "250", "800", "30000",
	) // TODO: configurable

	req, err := newRequest(searchValues)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	reader, err := charset.NewReader(resp.Body, "charset=ISO-8859-1")
	if err != nil {
		log.Fatal(err)
	}

	document, err := html.Parse(reader)
	if err != nil {
		log.Fatal(err)
	}

	items := scrape.FindAll(document, scrape.ByClass("aditem"))

	fmt.Println("PAGES:", pages(document))
	var ads []ad

	for _, item := range items {
		model, make := modelMake(getItemTitle(item))
		ads = append(ads, ad{
			Age:      getItemAge(item),
			Price:    getItemPrice(item),
			Year:     getItemYear(item),
			Kms:      getItemKms(item),
			Make:     make,
			Model:    model,
			Location: getItemLocation(item),
			URL:      getItemURL(item),
		})
	}

	return ads
}

func getNodeText(n *html.Node) string {
	var text string
	if n.FirstChild != nil {
		text = strings.TrimSpace(n.FirstChild.Data)
	}
	return text
}

func getItemTitle(itemNode *html.Node) string {
	titleNode, _ := scrape.Find(itemNode, scrape.ByClass("aditem-detail-title"))
	return getNodeText(titleNode)
}

func modelMake(title string) (string, string) {
	var model, make string
	mm := strings.Split(title, " - ")
	if len(mm) == 2 {
		model, make = mm[0], mm[1]
	}

	return model, make
}

func getItemPrice(itemNode *html.Node) string {
	priceNode, _ := scrape.Find(itemNode, scrape.ByClass("aditem-price"))
	return getNodeText(priceNode) + "€"
}

func getItemYear(itemNode *html.Node) string {
	yearNode, _ := scrape.Find(itemNode, scrape.ByClass("ano"))
	return getNodeText(yearNode)
}

func getItemKms(itemNode *html.Node) string {
	kmsNode, _ := scrape.Find(itemNode, scrape.ByClass("kms"))
	return getNodeText(kmsNode)
}

func getItemURL(itemNode *html.Node) string {
	detaillNode, _ := scrape.Find(itemNode, scrape.ByClass("aditem-detail-title"))
	return strings.Join([]string{baseURL, "motos-de-carretera", fmt.Sprint(detaillNode.Attr[0].Val)}, "/")
}

func getItemLocation(itemNode *html.Node) string {

	var location string

	locNode, ok := scrape.Find(itemNode, scrape.ByClass("x4"))
	if ok {
		locTxt := locNode.FirstChild.Data
		if strings.Contains(locTxt, "(") {
			location = strings.Split(locTxt, "(")[1]
			location = strings.TrimSuffix(location, ")")
		}
	}

	return location
}

func getItemAge(itemNode *html.Node) string {
	var ageTxt string

	ageNode, ok := scrape.Find(itemNode, scrape.ByClass("x6"))
	if ok {
		r := strings.NewReplacer("horas", "h", "hora", "h", "días", "d", "día", "d")
		ageTxt = ageNode.FirstChild.Data
		ageTxt = r.Replace(ageTxt)
	}

	return ageTxt
}

func getContent(node *html.Node) string {
	if node == nil {
		return ""
	}
	var content string

	childNode := node.FirstChild
	for ; childNode != nil; childNode = childNode.NextSibling {
		content = strings.Join([]string{content, childNode.Data}, " ")
	}

	return content
}

func pages(node *html.Node) int {
	pageCount := 1

	pageLinks := scrape.FindAll(node, scrape.ByClass("adlist-paginator-pagelink"))
	if len(pageLinks) > 1 {
		pageCount = len(pageLinks) - 1
	}

	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	pageSummary, ok := scrape.Find(node, scrape.ByClass("adlist-paginator-summary"))
	if !ok {
		return pageCount
	}

	text := strings.Split(pageSummary.Data, "de")[1]
	text = strings.TrimSpace(text)
	pageCount, _ = strconv.Atoi(text)
	return pageCount
}
