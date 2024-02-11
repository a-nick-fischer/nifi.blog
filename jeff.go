package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/1set/gut/yos"
	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
)

const INDEX_TEMPLATE_FILE = "index.html"
const PHOTOS_TEMPLATE_FILE = "photos.html"
const BLOG_TEMPLATE_FILE = "blog.html"

const FAVICON_PATH = "assets/favicon.ico"

const OUTPUT_DIR = "build"
const THUMBNAILS_DIR = "build/thumbnails"
const BLOG_ENTRIES_HTML_DIR = "build/blog"

const BLOG_ENTRIES_MD_DIR = "entries"
const PHOTOS_DIR = "photos"
const TEMPLATES_DIR = "templates"
const ASSETS_DIR = "assets"

const THUMBNAIL_WIDTH = 200

type Photo struct {
	Name      string
	Title     string
	Longitude float64
	Latitude  float64
}

type Article struct {
	Slug     string
	Tags     []string
	Title    string
	Summary  string
	Date     string
	HtmlBody string
}

func main() {
	dontGenerateThumbnails := flag.Bool("no-thumbnails", false, "Don't generate thumbnails")
	flag.Parse()

	articles := readArticles()
	photos := readPhotos()
	templates := readTemplates()

	regenerateOutputDir(*dontGenerateThumbnails)

	if !*dontGenerateThumbnails {
		generateThumbnails(photos)
	}

	generateBlogEntries(articles, templates)
	generateTemplate(templates, BLOG_TEMPLATE_FILE, articles)
	generateTemplate(templates, PHOTOS_TEMPLATE_FILE, photos)
	generateTemplate(templates, INDEX_TEMPLATE_FILE, "")
	copyFiles()
}

func regenerateOutputDir(dontDeleteThumbnails bool) {
	if yos.ExistDir(OUTPUT_DIR) {
		fmt.Println("Clearing build dir...")

		files, err := os.ReadDir(OUTPUT_DIR)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			filename := yos.JoinPath(OUTPUT_DIR, file.Name())

			if dontDeleteThumbnails && filename == filepath.Clean(THUMBNAILS_DIR) {
				continue
			}

			err := os.RemoveAll(filename)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		fmt.Println("Creating build dir...")

		err := os.MkdirAll(OUTPUT_DIR, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func generateTemplate(templates *template.Template, templateFile string, args any) {
	fmt.Printf("Generating %s...\n", templateFile)
	htmlOutputPath := yos.JoinPath(OUTPUT_DIR, templateFile)
	outputFile, err := os.Create(htmlOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	err = templates.ExecuteTemplate(outputFile, templateFile, args)
	if err != nil {
		log.Fatal(err)
	}
}

func copyFiles() {
	fmt.Println("Copying files...")
	err := yos.CopyFile(FAVICON_PATH, OUTPUT_DIR)
	if err != nil {
		log.Fatal(err)
	}

	err = yos.CopyDir(PHOTOS_DIR, yos.JoinPath(OUTPUT_DIR, PHOTOS_DIR))
	if err != nil {
		log.Fatal(err)
	}

	err = yos.CopyDir(ASSETS_DIR, yos.JoinPath(OUTPUT_DIR, ASSETS_DIR))
	if err != nil {
		log.Fatal(err)
	}
}

func generateThumbnails(photos []Photo) {
	fmt.Println("Generating thumbnails...")

	err := os.MkdirAll(THUMBNAILS_DIR, 0777)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(len(photos))

	for _, image := range photos {
		go func(image string) {
			defer wg.Done()
			generateThumbnail(image)
		}(image.Name)
	}

	wg.Wait()
}

func generateThumbnail(image string) {
	imagePath := yos.JoinPath(PHOTOS_DIR, image)
	src, err := imaging.Open(
		imagePath,
		imaging.AutoOrientation(true),
	)

	if err != nil {
		log.Fatal(err)
	}

	src = imaging.Resize(src, THUMBNAIL_WIDTH, 0, imaging.Lanczos)

	thumbnailPath := yos.JoinPath(THUMBNAILS_DIR, image)
	err = imaging.Save(src, thumbnailPath, imaging.JPEGQuality(80))
	if err != nil {
		log.Fatal(err)
	}
}

func generateBlogEntries(articles []Article, templates *template.Template) {
	fmt.Println("Generating blog entries...")

	err := os.MkdirAll(BLOG_ENTRIES_HTML_DIR, 0777)
	if err != nil {
		log.Fatal(err)
	}

	for _, article := range articles {

		htmlArticlePath := fmt.Sprintf("%s/%s.html", BLOG_ENTRIES_HTML_DIR, article.Slug)
		outputFile, err := os.Create(htmlArticlePath)
		if err != nil {
			log.Fatal(err)
		}

		err = templates.ExecuteTemplate(outputFile, "article.html", article)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func readTemplates() *template.Template {
	templateGlob := fmt.Sprintf("%s/*.html", TEMPLATES_DIR)

	templates := template.Must(
		template.ParseGlob(templateGlob),
	)

	return templates
}

func readArticles() []Article {
	rawArticles, err := os.ReadDir(BLOG_ENTRIES_MD_DIR)
	if err != nil {
		log.Fatal(err)
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(
			meta.Meta,
		),
	)

	articles := make([]Article, len(rawArticles))

	for i, article := range rawArticles {
		mdArticlePath := yos.JoinPath(BLOG_ENTRIES_MD_DIR, article.Name())

		source, err := os.ReadFile(mdArticlePath)
		if err != nil {
			log.Fatal(err)
		}

		context := parser.NewContext()
		var buf bytes.Buffer
		if err := markdown.Convert(source, &buf, parser.WithContext(context)); err != nil {
			log.Fatal(err)
		}

		basename := article.Name()
		filename := strings.TrimSuffix(basename, filepath.Ext(basename))

		metaData := meta.Get(context)

		tags := metaData["tags"].([]interface{})
		tagsStr := make([]string, len(tags))
		for i, tag := range tags {
			tagsStr[i] = tag.(string)
		}

		articles[i] = Article{
			Slug:     filename,
			Tags:     tagsStr,
			Title:    metaData["title"].(string),
			Summary:  metaData["summary"].(string),
			Date:     metaData["date"].(string),
			HtmlBody: buf.String(),
		}
	}

	return articles
}

func readPhotos() []Photo {
	images, err := os.ReadDir(PHOTOS_DIR)
	if err != nil {
		log.Fatal(err)
	}

	names := make([]Photo, len(images))
	for i, image := range images {
		imagePath := yos.JoinPath(PHOTOS_DIR, image.Name())

		f, err := os.Open(imagePath)
		if err != nil {
			log.Fatal(err)
		}

		lon := 0.0
		lat := 0.0

		info, err := exif.Decode(f)
		if err != nil {
			lon = -1.0
			lat = -1.0
		}

		longitudeExif, err := info.Get(exif.GPSLongitude)
		if err != nil {
			lon = -1.0
			lat = -1.0
		}

		latitudeExif, err := info.Get(exif.GPSLatitude)
		if err != nil {
			lon = -1.0
			lat = -1.0
		}

		if lon == -1.0 || lat == -1.0 {
			names[i] = Photo{
				Name:      image.Name(),
				Title:     image.Name(),
				Longitude: lon,
				Latitude:  lat,
			}
			continue
		}

		lon = exifGPSDataToCoordinate(longitudeExif)
		lat = exifGPSDataToCoordinate(latitudeExif)

		names[i] = Photo{
			Name:      image.Name(),
			Title:     image.Name(),
			Longitude: lon,
			Latitude:  lat,
		}
	}

	return names
}

func exifGPSDataToCoordinate(data *tiff.Tag) float64 {
	degreesRat, err := data.Rat(0)
	if err != nil {
		log.Fatal(err)
	}

	minutesRat, err := data.Rat(1)
	if err != nil {
		log.Fatal(err)
	}

	secondsRat, err := data.Rat(2)
	if err != nil {
		log.Fatal(err)
	}

	degrees, _ := degreesRat.Float64()
	minutes, _ := minutesRat.Float64()
	seconds, _ := secondsRat.Float64()

	return degrees + minutes/60 + seconds/3600
}