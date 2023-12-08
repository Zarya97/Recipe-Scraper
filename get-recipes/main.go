package main

//
// Imports
//

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)


type Nutrition struct {
	Name string `json:"name"`
	Amount int `json:"amount"`
	Unit string `json:"unit"`
}


type Time struct {
	Prep int `json:"prep"`
	Active int `json:"active"`
	Cook int `json:"cook"`
	Total int `json:"total"`
}

type Instruction struct {
	Step string `json:"step"`
}

type Ingredient struct {
	Name string `json:"name"`
}

type Tag struct {
	Tag string `json:"tag"`
}

// A structure for each recipe.
type Recipe struct {
	Id string `json:"id"`
	Url string `json:"url"`
	Title string `json:"title"`
	Author string `json:"author"`
	Description string `json:"description"`
	Level string `json:"level"`
	ImageUrl string `json:"imageUrl"`
	Time Time `json:"time"`
	Yield string `json:"yield"`
	Ingredients []Ingredient `json:"ingredients"`
	Instructions []Instruction `json:"instructions"`
	Nutrition []Nutrition `json:"nutrition"`
	Tags []Tag `json:"tags"`
}

//
// Simple constants to control speed if you get rate limited
//

// How many websites to download at a time
var BATCHSIZE = 25
// Number of seconds to sleep between batches
const SLEEPINTERVAL = 2

//
// Functions for loading strings, manipulating strings, data, etc.
//

// Load all lines from a file into a []string. and returns []string
func loadFileLines(filename string) []string {
	file, err := os.ReadFile(filename)

	if err != nil {
		log.Fatal(err)
	}

	content := string(file)
	lines := strings.Split(content, "\n")

	return lines

}

// Returns the MD5 sum of the entered string as a string
func hashString(str string) string {
	bytesHash := md5.Sum([]byte(str))
	stringHash := hex.EncodeToString(bytesHash[:])
	return stringHash
}

// Returns a string with only letters and numbers
func cleanString(s string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9 -]+")

	if err != nil {
		log.Println("Problem with cleaning the string", s, err)
	}

	cleanString := reg.ReplaceAllString(s, "")
	cleanString = strings.TrimSpace(cleanString)

	return cleanString
}

func extractStringTime(time string) int {
	// Remove square brackets and extra spaces
	time = strings.ReplaceAll(time, "[", "")
	time = strings.ReplaceAll(time, "]", "")
	time = strings.TrimSpace(time)

	times := strings.Split(time, " ")
	totalMinutes := 0

	// If there are no items in the slice, return -1
	if len(times) < 1 {
		log.Println("Invalid time string - of 0 length", times)
		return -1
	}

	// If there is only a number, return that number
	if len(times) == 1 {
		rtrTime, err := strconv.Atoi(times[0])
		if err != nil {
			log.Println("Something went wrong converting the time string", times, err)
			return -1
		}
		return rtrTime
	}

	if len(times)%2 != 0 {
		log.Println("Invalid time string - uneven length greater than 1:", times)
		return -1
	}

	for s := range times {

		if s == len(times)-1 {
			continue
		}

		if times[s+1] == "hr" {
			hours, err := strconv.Atoi(times[s])
			if err != nil {
				log.Println("Cannot convert hours to minutes", hours, err)
			}

			totalMinutes += hours * 60
		}

		if times[s+1] == "min" {
			minutes, err := strconv.Atoi(times[s])
			if err != nil {
				log.Println("Cannot convert minutes to minutes", minutes, err)
			}

			totalMinutes += minutes
		}
		continue
	}

	return totalMinutes
}


//
// Fetching the HTML
//

// Takes a URL and returns a goquery document object
func getUrlContent(url string) *goquery.Document {

	// HTTP get
	response, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}

	// Close the HTTP request later, to be polite
	defer response.Body.Close()

	// Turn the raw HTTP response from the site into a goquery response object
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Println("There was an error loading the HTTP body for URL", url, err)
	}

	return document

}

//
// Functions for getting data out of the html
//

// Get the title of the recipe
func getRecipeTitle(doc *goquery.Document) string {
	title := doc.Find("span[class=o-AssetTitle__a-HeadlineText]").First().Text()
	title = strings.TrimSpace(title)
	title = cleanString(title)
	return title
}

// Get the author of the recipe
func getRecipeAuthor(doc *goquery.Document) string {
	name := doc.Find("span[class=o-Attribution__a-Name] a").First().Text()
	name = cleanString(name)
	return name
}

// Get the description of the recipe
func getRecipeDescription(doc *goquery.Document) string {
	description := doc.Find("div[class=o-AssetDescription__a-Description]").First().Text()
	description = strings.TrimSpace(description)
	return description
}

// Get the level of the recipe
func getRecipeLevel(doc *goquery.Document) string {
	level := doc.Find("ul[class=o-RecipeInfo__m-Level] span[class=o-RecipeInfo__a-Description]").First().Text()
	level = strings.TrimSpace(level)
	return level
}

// Get the image URL of the recipe
func getRecipeImageUrl(doc *goquery.Document) string {
  var imageUrl string

  // Find the image element with the desired class
  img := doc.Find("div[class=m-MediaBlock__m-MediaWrap]").Children().First()

  imageUrl, _ = img.Attr("src")
	cleanUrl := imageUrl
	if len(imageUrl) > 3 {
		cleanUrl = imageUrl[2:]
	}

	pos := strings.LastIndex(cleanUrl, "/") + 1
  lastPart := cleanUrl[pos:]
	if lastPart == "1474463768097.jpeg" {
		return ""
	}

  return cleanUrl
}

func getRecipeTimes(doc *goquery.Document) (string, string, string, string) {
	prepTime := ""
	activeTime := ""
	cookTime := ""
	totalTime := ""
	doc.Find("ul.o-RecipeInfo__m-Time").Children().Each(func(index int, element *goquery.Selection) {
		headline := strings.TrimSpace(element.Text())
		if strings.Contains(headline, "Prep:") {
			prepTime = strings.Split(strings.TrimSpace(element.Text()), ":")[1]
			//prepTime = strings.Replace(strings.TrimSpace(prep), " ", " ", -1)
		} else if strings.Contains(headline, "Active:") {
			activeTime = strings.Split(strings.TrimSpace(element.Text()), ":")[1]
			//activeTime = strings.Replace(strings.TrimSpace(active), " ", " ", -1)
		} else if strings.Contains(headline, "Cook:") {
			cookTime = strings.Split(strings.TrimSpace(element.Text()), ":")[1]
			//cookTime = strings.Replace(strings.TrimSpace(cook), " ", " ", -1)
		} else if strings.Contains(headline, "Total:") {
			totalTime = strings.Split(strings.TrimSpace(element.Text()), ":")[1]
			//totalTime = strings.Replace(strings.TrimSpace(total), " ", " ", -1)
		}
	})

	doc.Find("ul.o-RecipeInfo__m-Level").Children().Each(func(index int, element *goquery.Selection) {
		headline := strings.TrimSpace(element.Text())
		if strings.Contains(headline, "Total:") {
			totalTime = strings.Split(strings.TrimSpace(element.Text()), ":")[1]
			//totalTime = strings.Replace(strings.TrimSpace(total), " ", "", -1)
		}
	})

	return prepTime, activeTime, cookTime, totalTime
}

// Gets the yeild and returns it as a string
func getRecipeYeild(doc *goquery.Document) string {
	yield := doc.Find("ul[class=o-RecipeInfo__m-Yield] li span[class=o-RecipeInfo__a-Description]").First().Text()
	yield = strings.TrimSpace(yield)
	return yield
}

// Gets the recipe nutrition and returns it as an array of Nutrition structs
func getRecipeNutrition(doc *goquery.Document) []Nutrition {
	nutrition := []Nutrition{}

	doc.Find("dl[class=m-NutritionTable__a-Content] dt").Each(func(index int, dt *goquery.Selection) {
		// Find the corresponding dd element
		dd := dt.Next()

		// Initialize a new Nutrition item
		nutritionItem := Nutrition{}

		// Set the name in the Nutrition item
		nutritionItem.Name = strings.TrimSpace(dt.Text())

		// Parse the amount and unit and set them in the Nutrition item
		text := strings.TrimSpace(dd.Text())
		parts := strings.SplitN(text, " ", 2)
		if len(parts) == 2 {
			nutritionItem.Amount, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
			nutritionItem.Unit = parts[1]
		}
		if len(parts) == 1 {
			nutritionItem.Amount, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
			nutritionItem.Unit = ""
		}

		// Append the Nutrition item to the slice
		nutrition = append(nutrition, nutritionItem)
	})

	return nutrition
}

// Gets the recipe ingredients and returns them as an array of (large) strings
func getRecipeIngredients(doc *goquery.Document) []Ingredient {

	ingredients := []Ingredient{}

	doc.Find("div[class=o-Ingredients__m-Body] p").Each(func(index int, element *goquery.Selection) {
		ingredient := Ingredient{}
		ingredient.Name = strings.TrimSpace(element.Text())
		ingredients = append(ingredients, ingredient)
	
	})

	if len(ingredients) < 1 {
		ingredients = []Ingredient{}
		return ingredients
	}

	// If we grab the select item, just remove it from the slice
	if ingredients[0].Name == "Deselect All" || ingredients[0].Name == "Select All" {
		ingredients = ingredients[1:]
	}

	return ingredients
}

// Gets the recipe directions and returns them as an array of (large) strings
func getRecipeInstructions(doc *goquery.Document) []Instruction {

	instructions := []Instruction{}

	doc.Find("li[class=o-Method__m-Step]").Each(func(index int, element *goquery.Selection) {
		instruction := Instruction{}
		instruction.Step = strings.TrimSpace(element.Text())
		instructions = append(instructions, instruction)
	
	})
	return instructions
}

func getRecipeTags(doc *goquery.Document) []Tag {
	tags := []Tag{}

	// Updated selector to include all classes
	body := doc.Find("div.o-Capsule__m-TagList.m-TagList a")
	body.Each(func(index int, element *goquery.Selection) {
		tag := Tag{}
		tag.Tag = strings.TrimSpace(element.Text())
		tags = append(tags, tag)
	})
	return tags
}



//
// Collecting, storing, writing the recipes, etc.
//

// Takes in a pointer to an empty Recipe struct
// And fills out every value.
func collectRecipe(recipeObj *Recipe, url string) *Recipe {

	// Set what we can set immediately
	recipeObj.Id = hashString(url)
	recipeObj.Url = url

	log.Println("Fetching recipe at", url)

	// Get goquery document from recipe URL
	doc := getUrlContent(url)

	// Get recipe title, author, etc
	recipeObj.Title = getRecipeTitle(doc)
	recipeObj.Author = getRecipeAuthor(doc)
	recipeObj.Description = getRecipeDescription(doc)
	recipeObj.Level = getRecipeLevel(doc)
	recipeObj.ImageUrl = getRecipeImageUrl(doc)

	prep, active, cook, total := getRecipeTimes(doc)
	recipeObj.Time.Prep = extractStringTime(prep)
	recipeObj.Time.Active = extractStringTime(active)
	recipeObj.Time.Cook = extractStringTime(cook)
	recipeObj.Time.Total = extractStringTime(total)

	// Yeild
	recipeObj.Yield = getRecipeYeild(doc)

	// Nutrition
	recipeObj.Nutrition = getRecipeNutrition(doc)

	// Ingredients
	recipeObj.Ingredients = getRecipeIngredients(doc)

	// Directions
	recipeObj.Instructions = getRecipeInstructions(doc)

	// Tags
	recipeObj.Tags = getRecipeTags(doc)

	getRecipeTimes(doc)

	log.Printf("Successfully collected recipe %s (%s)", recipeObj.Title, recipeObj.Id)

	return recipeObj

}


//
// Main type of goroutine that will handle each recipe
//

// Main goroutine that will handle each recipe. Will take in the recipe object
// and will send it to the recipe channel when done.
func recipeRoutine(recipeObj *Recipe, url string, wg *sync.WaitGroup, c chan Recipe) {
	defer wg.Done()

	// If the URL is invalid, then let the user know and continue
	if strings.HasPrefix(url, "https://www") {
		recipe := collectRecipe(recipeObj, url)
		c <- *recipe
	} else {
		log.Println("Invalid URL given to routine, ignoring:", url)
	}
	
}

func (r *Recipe) ToJSON() (string, error) {
	jsonData, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// Routine for writing to the file. Only one is spun up so as to avoid using a
// MutEx for a file.
func writerRoutine(c chan Recipe) {
	// Open the file for writing or create it if it doesn't exist
	f, err := os.OpenFile("./recipes.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		log.Fatal("Error trying to open the output file", err)
	}

	// Write the opening bracket for the JSON array
	f.WriteString("[")

	firstRecipe := true // Flag to determine if it's the first recipe

	for {
		select {
		case recipe := <-c:

			// Convert the recipe to JSON
			jsonData, err := recipe.ToJSON()
			if err != nil {
				log.Println("Error converting recipe to JSON", err)
				continue
			}

			// Add a comma before writing the JSON data (except for the first recipe)
			if !firstRecipe {
				f.WriteString(",")
				f.WriteString("\n")
			}

			// Write the JSON data to the file
			f.WriteString(jsonData)

			firstRecipe = false // Update the flag after writing the first recipe

		default:
			log.Println("No recipes in channel, waiting 2 seconds.")
			time.Sleep(2 * time.Second)
		}
	}

}



//
// Main
//

func main() {

	// Check for filename argument
	if len(os.Args) != 2 {
		fmt.Println("You need to pass a single filename for the links.")
		fmt.Println("Ex: ./get-fn-recipes recipeslinks.txt")
		os.Exit(1)
	}

	// Initialize
	log.Println("Starting up Foodnetwork Website Recipe Downloader v0.1")
	targetLinks := loadFileLines(os.Args[1])
	log.Println(fmt.Sprintf("Loaded a total of %d links from file %s", len(targetLinks), os.Args[1]))

	// Make a channel for our recipes between downloaders and the writer
	c := make(chan Recipe, 8)

	// Start the writer
	go writerRoutine(c)

	// Set up Synchronization for the recipe routines
	var wg sync.WaitGroup;
	wg.Add(len(targetLinks))

	// Iterate over all links in the given file, starting a goroutine for each one
	counter := 0
	for line := range targetLinks {
		newRecipe := &Recipe{}

		// Sometimes the URLs in the file pick up \r if on Windows >.>
		url := strings.ReplaceAll(targetLinks[line], "\r", "")
		url = fmt.Sprintf("https://%s", url)
		
		go recipeRoutine(newRecipe, url, &wg, c)
		counter += 1

		if counter >= BATCHSIZE {
			log.Println("Pausing for batch.")
			time.Sleep(SLEEPINTERVAL * time.Second)
			counter = 0
		}
	}

	// Wait until all go routines using wg are done.
	wg.Wait()

	log.Println("All recipe routines have finished. Waiting 5 seconds for writer routine.")
	time.Sleep(5 * time.Second)
	log.Println("Closing recipe routine channel.")

	close(c)

	// Indicate to the user that we are done and are going to take a nap now.
	log.Println("All recipes written to disk. Shutting down.")
}