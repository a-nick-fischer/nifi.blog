package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"text/template"

	"github.com/1set/gut/yos"
	"github.com/disintegration/imaging"
	"github.com/ikeikeikeike/go-sitemap-generator/v2/stm"
	figure "github.com/mangoumbrella/goldmark-figure"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
)

const INDEX_TEMPLATE_FILE = "index.html"
const PHOTOS_TEMPLATE_FILE = "photos.html"
const BLOG_TEMPLATE_FILE = "blog.html"

const FAVICON_PATH = "assets/favicon.svg"
const ROBOTSTXT_PATH = "assets/robots.txt"

const OUTPUT_DIR = "build"
const THUMBNAILS_DIR = "build/thumbnails"
const BLOG_ENTRIES_HTML_DIR = "build/blog"

const BLOG_ENTRIES_MD_DIR = "entries"
const PHOTOS_DIR = "photos"
const TEMPLATES_DIR = "templates"
const ASSETS_DIR = "assets"

const MAX_IMAGE_SIZE = 25 * 1024 * 1024
const THUMBNAIL_WIDTH = 200

var SORTING_PRIORITY = [2]string{
	"actually good", "ok",
}

type Photo struct {
	Name      string
	Title     string
	Tags      []string
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

	generatePagesFromTemplates(templates, articles, photos)

	generateBlogEntriesFromMarkdown(templates, articles)

	generateSitemap(photos, articles)

	copyAssetsToOutputDirectory()
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

func embedSvg(svgUrl string) string {
	fmt.Printf("Embedding SVG (%s)...\n", svgUrl)

	var body []byte

	if strings.HasPrefix(svgUrl, "http") {
		// Download the SVG
		resp, err := http.Get(svgUrl)
		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Fatalf("Failed to download %s: %s", svgUrl, resp.Status)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// Read the SVG from the local filesystem
		var err error
		body, err = os.ReadFile(svgUrl)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Remove newlines
	body = bytes.ReplaceAll(body, []byte("\n"), []byte(""))

	// Escape the # in the body
	kindaUrlEncoded := bytes.ReplaceAll(body, []byte("#"), []byte("%23"))

	return fmt.Sprintf("data:image/svg+xml;charset=utf-8,%s", kindaUrlEncoded)
}

func generateSitemap(photos []Photo, articles []Article) {
	fmt.Println("Generating sitemap...")

	sitemap := stm.NewSitemap(1)

	sitemap.SetVerbose(false)

	sitemap.SetDefaultHost("https://nifi.blog")

	sitemap.Create()

	imageUrls := make([]stm.URL, len(photos))
	for i, photo := range photos {
		// Yes we have to use the full URL here, because reasons...
		loc := fmt.Sprintf("https://nifi.blog/photos/%s", photo.Name)
		imageUrls[i] = stm.URL{{"loc", loc}}
	}

	for _, article := range articles {
		sitemap.Add(stm.URL{{"loc", fmt.Sprintf("/blog/%s", article.Slug)}})
	}

	sitemap.Add(stm.URL{{"loc", "/"}})
	sitemap.Add(stm.URL{{"loc", "/blog"}})
	sitemap.Add(stm.URL{{"loc", "/photos"}, {"image", imageUrls}})

	xml := sitemap.XMLContent()

	err := os.WriteFile(yos.JoinPath(OUTPUT_DIR, "sitemap.xml"), xml, 0777)
	if err != nil {
		log.Fatal(err)
	}
}

func generatePagesFromTemplates(templates *template.Template, articles []Article, photos []Photo) {
	generateTemplate(templates, BLOG_TEMPLATE_FILE, articles)
	generateTemplate(templates, PHOTOS_TEMPLATE_FILE, photos)
	generateTemplate(templates, INDEX_TEMPLATE_FILE, "")
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

func copyAssetsToOutputDirectory() {
	fmt.Println("Copying files...")
	err := yos.CopyFile(FAVICON_PATH, OUTPUT_DIR)
	if err != nil {
		log.Fatal(err)
	}

	err = yos.CopyFile(ROBOTSTXT_PATH, OUTPUT_DIR)
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

func generateBlogEntriesFromMarkdown(templates *template.Template, articles []Article) {
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
		template.New("main").
			Funcs(template.FuncMap{"embedSvg": embedSvg}).
			ParseGlob(templateGlob),
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
			figure.Figure,
			emoji.Emoji,
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

	photos := make([]Photo, len(images))
	for i, image := range images {
		imagePath := yos.JoinPath(PHOTOS_DIR, image.Name())

		// Needed because we cannot serve files larger than 25MB from Cloudflare
		fileInfo, err := os.Stat(imagePath)
		if err != nil {
			log.Fatal(err)
		}

		if fileInfo.Size() > MAX_IMAGE_SIZE {
			log.Fatalf("Image %s is too large", image.Name())
		}

		f, err := os.Open(imagePath)
		if err != nil {
			log.Fatal(err)
		}

		info, err := exif.Decode(f)
		if err != nil {
			log.Fatalf("Error decoding exif data for %s: %s", image.Name(), err)
		}

		tags := extractKeywords(info)

		lat, lon, err := info.LatLong()
		if err != nil {
			lat = -1
			lon = -1
		}

		photos[i] = Photo{
			Name:      image.Name(),
			Title:     image.Name(),
			Tags:      tags,
			Longitude: lon,
			Latitude:  lat,
		}
	}

	slices.SortFunc(photos, customSortImagesAccordingToMyFlawlessLogic)

	return photos
}

func extractKeywords(info *exif.Exif) []string {
	tag, err := info.Get(exif.XPKeywords)
	if err != nil {
		log.Fatalf("Error extracting keywords: %s", err)
	}

	tags := strings.Split(string(tag.Val), ";")

	// For some god damned reason reading the keywords from the exif data
	// results in a bunch of null bytes in the string which we have to remove
	for i, tag := range tags {
		tags[i] = strings.ReplaceAll(tag, "\x00", "")
	}

	return tags
}

func customSortImagesAccordingToMyFlawlessLogic(a, b Photo) int {
	grade := func(photo Photo) int {
		for i, priority := range SORTING_PRIORITY {
			for _, tag := range photo.Tags {
				if tag == priority {
					return i
				}
			}
		}

		return len(SORTING_PRIORITY)
	}

	return grade(a) - grade(b)
}
