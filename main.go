package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Output struct {
	Url   string `json:"url"`
	Title string `json:"title"`
}

type Structured struct {
	Display []string `json:"display"`
	Like    int      `json:"likes"`
	Match   int      `json:"match"`
	Rated   bool     `json:rated"`
}

func main() {
	//fetch comment details, write to a file
	fetch()
	//parse file to get timestamps, populate a csv
	parse()
}

func parse() {
	b, err := os.ReadFile("output.json")
	if err != nil {
		log.Fatal(err)
	}
	overall := make(map[string][]*Comment)
	json.Unmarshal(b, &overall)
	final := make(map[string]*Structured)
	rated := make(map[string][]*Output)
	for id, comments := range overall {
		maxCnt := 0
		ratingFound := false
		var maxCmnt Comment
		for _, comment := range comments {
			cnt := strings.Count(comment.Items[0].Snippet.Display, id)
			if cnt >= maxCnt {
				maxCnt = cnt
				maxCmnt = *comment
				maxCmnt.Items[0].Snippet.Match = maxCnt
				maxCmnt.Items[0].Snippet.Display = strings.Replace(maxCmnt.Items[0].Snippet.Display, "\u0026amp;", "&", -1)
				maxCmnt.Items[0].Snippet.Display = strings.Replace(maxCmnt.Items[0].Snippet.Display, "\u003ca", " ", -1)

				//parse
				maxCmnt.Items[0].Snippet.ParsedDisplay = strings.Split(maxCmnt.Items[0].Snippet.Display, "\u003cbr\u003e")
				maxCmnt.Items[0].Snippet.ParsedOriginal = strings.Split(maxCmnt.Items[0].Snippet.Original, "\n")
				for index, str := range maxCmnt.Items[0].Snippet.ParsedDisplay {
					for i := 10; i >= 1; i-- {
						key := strconv.Itoa(i) + "/10"
						title := maxCmnt.Items[0].Snippet.ParsedOriginal[index]
						if strings.Contains(str, key) || strings.Contains(str, "("+strconv.Itoa(i)+")") {
							ratingFound = true
							if _, ok := rated[key]; !ok {
								rated[key] = make([]*Output, 0)
							}
							url := strings.Replace(str, `  href=\"`, " ", -1)
							if len(strings.Split(url, `"`)) > 1 {
								url = strings.Split(url, `"`)[1]
							}
							comment.Items[0].Snippet.ParsedUrl = url
							// title := str
							if len(strings.Split(str, `</a>`)) > 1 {
								title = strings.Split(str, `</a>`)[1]
							}
							title = strings.Replace(title, "-", " ", -1)
							title = strings.Replace(title, ",", " ", -1)
							title = strings.TrimSpace(title)
							actual := strings.Replace(title, " ", "", -1)
							if len(actual) < 0 {
								title = str
							}
							rated[key] = append(rated[key], &Output{
								Url:   url,
								Title: title,
							})
						}
					}

				}
			}
		}
		if maxCnt > 0 {
			var parsed []string
			for _, str := range maxCmnt.Items[0].Snippet.ParsedDisplay {
				str := strings.Replace(str, `href=\"`, " ", -1)
				str = strings.Replace(str, `href=`, " ", -1)
				str = strings.Replace(str, "-", " ", -1)
				str = strings.Replace(str, "<a>", " ", -1)
				str = strings.Replace(str, "</a>", " ", -1)
				str = strings.Replace(str, ">", " ", -1)

				str = strings.Replace(str, ",", " ", -1)
				str = strings.TrimSpace(str)
				parsed = append(parsed, str)
			}

			final[id] = &Structured{
				Rated:   ratingFound,
				Display: parsed,
				Like:    maxCmnt.Items[0].Snippet.Like,
				Match:   maxCmnt.Items[0].Snippet.Match,
			}
		}
	}

	by, err := json.Marshal(final)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(by))

	csvFile, err := os.Create("out.csv")
	csvwriter := csv.NewWriter(csvFile)
	for rating, content := range rated {
		for _, output := range content {
			var line []string
			line = append(line, strings.Split(rating, `/`)[0])
			line = append(line, output.Title)
			line = append(line, output.Url)
			csvwriter.Write(line)
		}
	}
	csvwriter.Flush()

	csvFile2, err := os.Create("raw.csv")
	csvwriter2 := csv.NewWriter(csvFile2)
	for id, content := range final {
		var line []string
		line = append(line, id)
		line = append(line, strconv.Itoa(content.Like))
		line = append(line, strconv.Itoa(content.Match))
		line = append(line, strconv.FormatBool(content.Rated))

		line = append(line, strings.Join(content.Display, " "))
		csvwriter2.Write(line)

	}
	csvwriter2.Flush()
}

// fetch comments for video ids from list
func fetch() {
	file, err := os.Open("list")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	overall := make(map[string][]*Comment)
	for scanner.Scan() {
		id := scanner.Text()
		overall[id] = make([]*Comment, 0)
		comments, err := comments(id)
		if err != nil {
			panic(err)
		}
		fmt.Println(id)
		for _, comm := range comments.Items {
			comment, err := content(id, comm.Id)
			if err != nil {
				panic(err)
			}
			by, err := json.Marshal(overall)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(by))
			overall[id] = append(overall[id], comment)
		}
	}
	by, err := json.Marshal(overall)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(by))
}

var (
	key = "AIzaSyClEMPt6p0AAQiT9lkeKD14ZkXjx7FCAT0"
)

type CommentList struct {
	Items []Item `json:"items"`
}

type Item struct {
	Id string `json:"id"`
}

// curl "https://www.googleapis.com/youtube/v3/commentThreads?key=&videoId=1_JABJmSunM"
func comments(vId string) (*CommentList, error) {
	resp, err := http.Get(fmt.Sprintf("https://www.googleapis.com/youtube/v3/commentThreads?order=relevance&key=%s&videoId=%s", key, vId))
	if err != nil {
		return nil, err
	}
	var comments CommentList
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&comments)
	if err != nil {
		return nil, err
	}
	return &comments, nil
}

type Comment struct {
	Items []CommentData `json:"items"`
}

type CommentData struct {
	Id      string  `json:"id"`
	Snippet Snippet `json:"snippet"`
}

type Snippet struct {
	Display        string   `json:"textDisplay"`
	Original       string   `json:"textOriginal"`
	Like           int      `json:"likeCount"`
	Author         string   `json:"authorDisplayName"`
	Rating         string   `json:"viewerRating"`
	Match          int      `json:"match"`
	ParsedDisplay  []string `json:"parsedDisplay"`
	ParsedOriginal []string `json:"parsedOriginal"`
	ParsedUrl      string   `json:"url"`
}

func content(vid, cid string) (*Comment, error) {
	resp, err := http.Get(fmt.Sprintf("https://www.googleapis.com/youtube/v3/comments?&part=snippet&key=%s&videoId=%s&id=%s", key, vid, cid))
	if err != nil {
		return nil, err
	}
	var comment Comment
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&comment)
	if err != nil {
		return nil, err
	}
	return &comment, nil
}
