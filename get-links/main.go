package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"strings"
	"strconv"
	"os"

	"github.com/PuerkitoBio/goquery"
)

// Make a global object for handling URLs. This will allow the goroutines to write to it as
// The memory persists between scopes.
type recipes struct {
	urls []string
}

// Gets all of the recipes from a www.foodnetwork.com/recipes/recipes-a-z/ call.
// Will return all recipe URLs on that single page.
func getAllRecipeUrls(url string, recipeUrls *recipes, mut sync.Mutex, wg *sync.WaitGroup) {

	// HTTP get
	log.Println("Fetching URL", url)
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	// Close the HTTP request later, to be polite
	defer response.Body.Close()

	// Turn the raw HTTP response from the site into a goquery response object
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal("There was an error loading the HTTP body!", err)
	}

	// Find all of the links, and execute getLinkDetails
	// li[class=m-PromoList__a-ListItem] finds all `li` with class `.class=m-PromoList__a-ListItem`
	document.Find("li[class=m-PromoList__a-ListItem] a").Each(func(index int, element *goquery.Selection) {

		returnStr := "" // Define return string for manipulation

		// Extract the target element, href, and check if it exists
		href, exists := element.Attr("href")
		if exists != true {
			log.Println("An element did not actually have an HREF")
		}

		returnStr = href

		if href[:2] == "//" {
			returnStr = href[2:]
		}

		mut.Lock()
		recipeUrls.urls = append(recipeUrls.urls, returnStr)
		mut.Unlock()

		

	})

	wg.Done()
}


func getTotalPagesForLetter(url string) int {
	
	totalPages := 1

	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	// Close the HTTP request later, to be polite
	defer response.Body.Close()

	// Turn the raw HTTP response from the site into a goquery response object
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal("There was an error loading the HTTP body!", err)
	}

	document.Find("li[class=o-Pagination__a-ListItem]").Each(func(index int, element *goquery.Selection) {

		// Extract the target element, href, and check if it exists
		v, _ := strconv.Atoi(element.Text())

		if v > totalPages {
			totalPages = v
		}

	})

	logLine := fmt.Sprintf("The letter '%s' has a total of %d pages", url, totalPages)
	log.Println(logLine)

	return totalPages
}

func writeUrlsToFile(recipeUrls *recipes) {
	// Open the output file
	logFile := "linksout.txt"
	f, err := os.Create(logFile)
	if err != nil {
		log.Fatal(err)
		f.Close()
	}

	for _, s := range recipeUrls.urls {
		fmt.Fprintln(f, s)
	}

	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Links successfully saved to", logFile)
}



// Main
func main() {

	// Initiate the application, with logging and globals
	log.Println("Starting www.foodnetwork.com Recipe Downloader v0.1")

	apiAlphabet := "123,a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p,q,r,s,t,u,v,w,xyz"

	// Make a WaitGroup to handle the goroutines for getting the links
	var wg sync.WaitGroup

	// Make a MutEx for the url get
	var urlmut sync.Mutex

	// Make a pointer to our struct that will hold all of our recipes
	recipes := &recipes{}

	// Query the website for each letter of the alphabet that the website supports.
	for _, letter := range strings.Split(apiAlphabet, ",") {

		totalPages := getTotalPagesForLetter(fmt.Sprintf("https://www.foodnetwork.com/recipes/recipes-a-z/%s", letter))

		// Add to the WaitGroup the number of pages we need to query for this letter
		wg.Add(totalPages)
		
		// Iterate over the pages and get all of the links, adding them to the global
		// recipes struct
		for i:=1; i<=totalPages; i++ {
			targetUrl := fmt.Sprintf("https://www.foodnetwork.com/recipes/recipes-a-z/%s/p/%d", letter, i)
			go getAllRecipeUrls(targetUrl, recipes, urlmut, &wg)
		}
	}
		
	// Wait for our 
	wg.Wait()

	writeUrlsToFile(recipes)
}