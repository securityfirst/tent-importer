package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/securityfirst/tent/repo/component"
)

var (
	cleanAlphaNum = regexp.MustCompile("[^[:alpha:]\\s\\d]")
	spaceTrim     = regexp.MustCompile("\\s+")
)

func MakeId(v string) string {
	v = cleanAlphaNum.ReplaceAllString(v, " ")
	v = spaceTrim.ReplaceAllString(v, "-")
	v = strings.ToLower(v)
	return v
}

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("%s <src> <dst>\n", os.Args[0])
		os.Exit(1)
	}
	dir := os.Args[1]
	dest := os.Args[2]
	log.Printf("Source: %s", dir)
	log.Printf("Destination: %s", dest)
	start := time.Now()
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	var cats = make(map[string]*component.Category)
	for _, f := range files {
		transform(cats, dir, f)
	}
	for _, c := range cats {
		if err := writeCmp(dest, c); err != nil {
			panic(err)
		}
		for _, s := range c.Subcategories() {
			sub := c.Sub(s)
			writeCmp(dest, sub)
			writeCmp(dest, sub.Checks())
			for _, item := range sub.Items() {
				writeCmp(dest, &item)
			}
		}
	}
	log.Printf("Successful created %d categories in %s", len(cats), time.Since(start))
}

func writeCmp(base string, c component.Component) error {
	path := filepath.Join(base, c.Path())
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(c.Contents()); err != nil {
		return err
	}
	return nil
}

func transform(categories map[string]*component.Category, dir string, f os.FileInfo) {
	locale := f.Name()
	dir = filepath.Join(dir, f.Name())
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if n := f.Name(); !strings.HasSuffix(n, ".json") || n == "strings.json" {
			continue
		}
		name := filepath.Join(dir, f.Name())
		file, err := os.Open(name)
		if err != nil {
			log.Fatal(err)
		}
		var list []struct {
			Title       string `json:"title`
			Body        string `json:"body`
			Text        string `json:"text`
			Category    string `json:"category`
			Subcategory string `json:"subcategory`
			Difficulty  string `json:"difficulty`
			NoCheck     bool   `json:""nocheck"`
		}
		if err := json.NewDecoder(file).Decode(&list); err != nil {
			log.Println(name, err)
		}
		for _, v := range list {
			// category
			catId := MakeId(v.Category)
			if _, ok := categories[catId]; !ok {
				categories[catId] = &component.Category{
					Id:     catId,
					Locale: locale,
					Name:   v.Category,
					Order:  float64(len(categories)),
				}
			}
			cat := categories[catId]
			// subcategory
			subId := MakeId(v.Subcategory)
			sub := cat.Sub(subId)
			if sub == nil {
				sub = &component.Subcategory{
					Id:    subId,
					Name:  v.Subcategory,
					Order: float64(len(cat.Subcategories())),
				}
				cat.Add(sub)
			}
			// check
			if v.Title == "" {
				sub.AddChecks(component.Check{
					Text:       v.Text,
					Difficulty: v.Difficulty,
					NoCheck:    v.NoCheck,
				})
				continue
			}
			// item
			var item = component.Item{
				Id:         MakeId(v.Title),
				Title:      v.Title,
				Body:       v.Body,
				Difficulty: v.Difficulty,
				Order:      float64(len(sub.Items())),
			}
			baseName := item.Id
			for i := 0; ; i++ {
				if err := sub.AddItem(&item); err != nil {
					item.Id = fmt.Sprintf("%s-%d", baseName, i)
					continue
				}
				break
			}
		}
	}
}
