package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type EpisodeInfo struct {
	Folder        string
	FileName      string
	AnimeTitle    string
	EpisodeNumber string
	Parser        string
}

type FilenamePattern struct {
	Pattern  *regexp.Regexp
	Function func(*EpisodeInfo) bool
}

var MEDIA_EXTENSIONS = []string{".MKV", ".OGM", ".AVI"}

func main() {
	print("Anime Indexer 0.0.1")
	if len(os.Args) > 1 {
		os.Chdir(os.Args[1])
	}
	allFiles, unknown_extensions := findAllMediaFiles()
	fmt.Printf("Total files found: %d \n", len(allFiles))
	fmt.Printf("Unknown extensions: %s \n", unknown_extensions)
	episodes := parseFileNames(allFiles)
	saveAnalysis(episodes)
}

func saveAnalysis(episodes []EpisodeInfo) {
	csvFile, err := os.Create("anime-index.csv")
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer csvFile.Close()
	csvwriter := csv.NewWriter(csvFile)
	csvwriter.Write([]string{"Anime Title", "Episode Number", "Parser", "Filename"})
	for _, episode := range episodes {
		_ = csvwriter.Write([]string{episode.AnimeTitle, episode.EpisodeNumber, episode.Parser, episode.FileName})
	}
	csvwriter.Flush()
}

// (^[^\[].*)( - )([a-zA-Z\d]+)( - )([^\[]+)([^\.]+)(\.)([a-zA-Z]+)
// .Hack  Sign - 01 - Role Play [Ahq](189Ff5e0)[Anidb]-1.mkv
func pattern1(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "1"
	regex := regexp.MustCompile(`(?mU)(^[^\[].*)( - )([a-zA-Z\d]+)( - )([^\[]+)([^\.]+)(\.)([a-zA-Z]+)`)
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	episodeInfo.AnimeTitle = res[0][1]
	episodeInfo.EpisodeNumber = res[0][3]
	return true
}

// ^(\[[^\]]+\])([^(]+)(\([\d]+[pP]\))([^\.]+)(\.)([a-zA-Z]+)
// [SubsPlease] 16bit Sensation - Another Layer - 01 (1080p) [C13E9494].mkv
// [Subsplease] Arifureta Shokugyou De Sekai Saikyou S2 - Ova P1 (1080P) [Ac47c50b].mkv
// Burn the Witch - #0.8 from [SubsPlease] Burn the Witch - #0.8 (1080p) [6CE13449].mkv
func pattern2(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "2"
	regex := regexp.MustCompile(`^(\[[^\]]+\])([^(]+)(\([\d]+[pP]\))([^\.]+)(\.)([a-zA-Z]+)`)
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	subpart := strings.TrimSpace(res[0][2])
	subpartRegex := regexp.MustCompile("(.*)(\\ -\\ )([a-zA-Z#\\ \\.\\d]+$)")
	resSubpart := subpartRegex.FindAllStringSubmatch(subpart, 1)
	if len(resSubpart) > 0 {
		episodeInfo.AnimeTitle = resSubpart[0][1]
		episodeInfo.EpisodeNumber = resSubpart[0][3]
	} else {
		episodeInfo.AnimeTitle = subpart
	}

	return true
}

// ^(\[[^\]]+\])([^(]+)(\([^\.]+)(\.)([a-zA-Z]+)
// [Doki] A Channel +Smile - 01 (1920X1080 Blu-Ray H264) [98223321]-1.mkv
func pattern3(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "3"
	var year string

	regex := regexp.MustCompile(`(\([12]\d\d\d\))`)
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	if len(res) > 0 {
		year = res[0][1]
	}

	regex = regexp.MustCompile(`^(\[[^\]]+\])([^(]+)(\([^\.]+)(\.)([a-zA-Z]+)`)
	res = regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	subpart := strings.TrimSpace(res[0][2])
	subpartRegex := regexp.MustCompile(`(.*)(\ -\ )([a-zA-Z\.\d]+$)`)
	resSubpart := subpartRegex.FindAllStringSubmatch(subpart, 1)
	if len(resSubpart) > 0 {
		episodeInfo.AnimeTitle = resSubpart[0][1]
		episodeInfo.EpisodeNumber = resSubpart[0][3]
	} else {
		subpartRegex = regexp.MustCompile(`(?mU)(.*)([a-zA-Z\.\d]+$)`)
		resSubpart = subpartRegex.FindAllStringSubmatch(subpart, 1)

		if len(resSubpart) > 0 {
			episodeInfo.AnimeTitle = resSubpart[0][1]
			episodeInfo.EpisodeNumber = resSubpart[0][2]
		} else {
			episodeInfo.AnimeTitle = subpart
		}
	}

	if (year != "") && (!strings.Contains(episodeInfo.AnimeTitle, year)) {
		episodeInfo.AnimeTitle = episodeInfo.AnimeTitle + " " + year
	}

	return true
}

// (^[^\[]+)(\ -\ )([^\.]+)(\.)([a-zA-Z]+)
// The Girl In Twilight - 01 - [Horriblesubs](1920X1080 H264)[9334C2b8]-1.mkv
// Akira  - [Thora](1888X1016 Blu-Ray H264)[B8fdce8a]-1.mkv
func pattern4(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "4"
	regex := regexp.MustCompile("(^[^\\[]+)(\\ -\\ )([^\\.]+)(\\.)([a-zA-Z]+)")
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	subpart := strings.TrimSpace(res[0][1])
	subpartRegex := regexp.MustCompile("(.*)(\\ -\\ )([a-zA-Z\\.\\d]+$)")
	resSubpart := subpartRegex.FindAllStringSubmatch(subpart, 1)
	if len(resSubpart) > 0 {
		episodeInfo.AnimeTitle = resSubpart[0][1]
		episodeInfo.EpisodeNumber = resSubpart[0][3]
	} else {
		episodeInfo.AnimeTitle = subpart
	}

	return true
}

// (^\[[^\]]+\])(.+)(\([\dpP]+\)[^\.]+)(\.)([a-zA-Z]+)$

// (^\[[^\]]+\])(.+)(\[[\dpP]+\][-\d]*)(\.)([a-zA-Z]+)$
// [Horriblesubs] Arifureta Shokugyou De Sekai Saikyou - 01 [1080P].mkv
// [Horriblesubs] Arifureta Shokugyou De Sekai Saikyou - 02 [1080P]-1.mkv
// [Subsplease] Arifureta Shokugyou De Sekai Saikyou S2 - Ova P1 (1080P) [Ac47c50b].mkv
func pattern5(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "5"
	regex := regexp.MustCompile("(^\\[[^\\]]+\\])(.+)(\\[[\\dpP]+\\][-\\d]*)(\\.)([a-zA-Z]+)$")
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	subpart := strings.TrimSpace(res[0][2])
	subpartRegex := regexp.MustCompile(`(.*)(\ -\ )([a-zA-Z\.\d ]+)`)
	resSubpart := subpartRegex.FindAllStringSubmatch(subpart, 1)
	if len(resSubpart) > 0 {
		episodeInfo.AnimeTitle = resSubpart[0][1]
		episodeInfo.EpisodeNumber = resSubpart[0][3]
	} else {
		episodeInfo.AnimeTitle = subpart
	}

	return true
}

// /(^\[[^\]]+\])(.+)([^ ]+)(\.)([a-zA-Z]+$)/gmU
// [Nep Blanc] Clockwork Planet 01.mkv
func pattern6(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "6"
	regex := regexp.MustCompile(`(?mU)(^\[[^\]]+\])(.*)\.([a-zA-Z]+$)`)
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	title := strings.TrimSpace(res[0][2])

	regex = regexp.MustCompile(`(^[^\[]+)`)
	res = regex.FindAllStringSubmatch(title, 1)
	title = strings.TrimSpace(res[0][1])

	regex = regexp.MustCompile(`(?mU)(.*)([\d\.]+$)`)
	res = regex.FindAllStringSubmatch(title, 1)
	if len(res) == 0 {
		episodeInfo.AnimeTitle = title
		return true
	}

	episodeInfo.EpisodeNumber = strings.TrimSpace(res[0][2])
	title = strings.TrimSpace(res[0][1])

	regex = regexp.MustCompile(`(?mU)(.*)( ?-?$)`)
	res = regex.FindAllStringSubmatch(title, 1)
	episodeInfo.AnimeTitle = strings.TrimSpace(res[0][1])
	return true
}

// (?mU)(.*)(\ -\ )[^ ]+(\ -\ )(.*)([a-zA-Z]+$)
// Telepathy Shoujo Ran - 26 - [M.3.3.W](1280X720 H264)[A086cdb2]-26.mkv
func pattern7(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "7"
	regex := regexp.MustCompile(`(?mU)(.*)(\ -\ )([^ ]+)(\ -\ )(.*)([a-zA-Z]+$)`)
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	episodeInfo.AnimeTitle = strings.TrimSpace(res[0][1])
	episodeInfo.EpisodeNumber = strings.TrimSpace(res[0][3])
	return true
}

// (^\([^\)]+\))(_)([^\(]+)(.*)$
// (Hi10)_Rurouni_Kenshin_-_64_(480p)_(DragonFox).mkv
func pattern8(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "8"
	regex := regexp.MustCompile(`(?m)(^\([^\)]+\))(_)([^\(]+)(.*)$`)
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	subpart := strings.TrimSpace(strings.ReplaceAll(res[0][3], "_", " "))
	subpartRegex := regexp.MustCompile(`(^.*)(\ -\ )([^ ]+$)`)
	resSubpart := subpartRegex.FindAllStringSubmatch(subpart, 1)
	if len(resSubpart) > 0 {
		episodeInfo.AnimeTitle = resSubpart[0][1]
		episodeInfo.EpisodeNumber = resSubpart[0][3]
	} else {
		episodeInfo.AnimeTitle = subpart
	}

	return true
}

// (^[\d]+)(.)([^\[]+)(.*)
// 04. Banner Of The Stars Ii (Seikai No Senki Ii) [Dvd 480P Hi10p Aac Ac3 Dual-Audio][Kuchikirukia]-4.mkv
func pattern9(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "9"
	regex := regexp.MustCompile(`(^[\d]+)(.)([^\[]+)(.*)`)
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	episodeInfo.AnimeTitle = strings.TrimSpace(res[0][3])
	episodeInfo.EpisodeNumber = strings.TrimSpace(res[0][1])
	return true
}

// (.*)(.)([\d]+)(.)(1080[Pp])(.*$)
// Evangelion.3.0+1.01.Thrice.Upon.A.Time.2021.1080P.Amzn.Web-Dl.Dd+.5.1.H.264-Rmb-1.mkv
func pattern10(episodeInfo *EpisodeInfo) bool {
	episodeInfo.Parser = "10"
	regex := regexp.MustCompile(`(.*)(.)([\d]+)(.)(1080[Pp])(.*$)`)
	res := regex.FindAllStringSubmatch(episodeInfo.FileName, 1)
	episodeInfo.AnimeTitle = strings.TrimSpace(res[0][1])
	return true
}

func compilePatterns() []FilenamePattern {
	var result []FilenamePattern

	result = append(result, FilenamePattern{
		regexp.MustCompile(`(^[^\[].*)( - )([a-zA-Z\d]+)( - )([^\[]+)([^\.]+)(\.)([a-zA-Z]+)`), // .Hack  Sign - 01 - Role Play [Ahq](189Ff5e0)[Anidb]-1.mkv
		pattern1})
	result = append(result, FilenamePattern{
		regexp.MustCompile("^(\\[[^\\]]+\\])([^(]+)(\\([\\d]+[pP]\\))([^\\.]+)(\\.)([a-zA-Z]+)"), // [SubsPlease] 16bit Sensation - Another Layer - 01 (1080p) [C13E9494].mkv
		pattern2})
	result = append(result, FilenamePattern{
		regexp.MustCompile("^(\\[[^\\]]+\\])([^(]+)(\\([^\\.]+)(\\.)([a-zA-Z]+)"), // [Doki] A Channel +Smile - 01 (1920X1080 Blu-Ray H264) [98223321]-1.mkv
		pattern3})
	result = append(result, FilenamePattern{
		regexp.MustCompile("(^[^\\[]+)(\\ -\\ )([^\\.]+)(\\.)([a-zA-Z]+)"), // The Girl In Twilight - 01 - [Horriblesubs](1920X1080 H264)[9334C2b8]-1.mkv
		pattern4})
	result = append(result, FilenamePattern{
		regexp.MustCompile("(^\\[[^\\]]+\\])(.+)(\\[[\\dpP]+\\][-\\d]*)(\\.)([a-zA-Z]+)$"), // [Horriblesubs] Arifureta Shokugyou De Sekai Saikyou - 01 [1080P].mkv
		pattern5})
	result = append(result, FilenamePattern{
		regexp.MustCompile(`(?mU)(^\[[^\]]+\])(.*)\.([a-zA-Z]+$)`), // [Nep Blanc] Clockwork Planet 01.mkv
		pattern6})
	result = append(result, FilenamePattern{
		regexp.MustCompile(`(?mU)(.*)(\ -\ )([^ ]+)(\ -\ )(.*)([a-zA-Z]+$)`), // Telepathy Shoujo Ran - 26 - [M.3.3.W](1280X720 H264)[A086cdb2]-26.mkv
		pattern7})
	result = append(result, FilenamePattern{
		regexp.MustCompile(`(?m)(^\([^\)]+\))(_)([^\(]+)(.*)$`), // Telepathy Shoujo Ran - 26 - [M.3.3.W](1280X720 H264)[A086cdb2]-26.mkv
		pattern8})
	result = append(result, FilenamePattern{
		regexp.MustCompile(`(^[\d]+)(.)([^\[]+)(.*)`), // 04. Banner Of The Stars Ii (Seikai No Senki Ii) [Dvd 480P Hi10p Aac Ac3 Dual-Audio][Kuchikirukia]-4.mkv
		pattern9})
	result = append(result, FilenamePattern{
		regexp.MustCompile(`(.*)(.)([\d]+)(.)(1080[Pp])(.*$)`), // Evangelion.3.0+1.01.Thrice.Upon.A.Time.2021.1080P.Amzn.Web-Dl.Dd+.5.1.H.264-Rmb-1.mkv
		pattern10})

	return result
}

func parseFileNames(fileNames []string) []EpisodeInfo {
	var patterns = compilePatterns()
	var result = []EpisodeInfo{}
	var parsed int
	var unparsed int
	for _, fileName := range fileNames {
		var episode EpisodeInfo
		episode.FileName = filepath.Base(fileName)
		episode.Folder = filepath.Dir(fileName)
		if parseEpisodeName(&episode, patterns) {
			parsed += 1
			optimizeParsing(&episode)
		} else {
			unparsed += 1
		}
		result = append(result, episode)
	}
	fmt.Printf("Parsed filenames: %d\n", parsed)
	fmt.Printf("Unparsed filenames: %d\n", unparsed)
	return result
}

func optimizeParsing(episode *EpisodeInfo) {
	regex := regexp.MustCompile(`(.*)([vV]\d+$)`)
	if regex.MatchString(episode.EpisodeNumber) {
		res := regex.FindAllStringSubmatch(episode.EpisodeNumber, 1)
		episode.EpisodeNumber = strings.TrimSpace(res[0][1])
	}

	regex = regexp.MustCompile(`[^_](_)[^_]|^(_)|(_)$`)
	res := regex.FindAllStringSubmatchIndex(episode.AnimeTitle, -1)

	for i := range res {
		if res[i][2] == -1 {
			episode.AnimeTitle = replaceAtIndex(episode.AnimeTitle, ' ', res[i][0])
		} else {
			episode.AnimeTitle = replaceAtIndex(episode.AnimeTitle, ' ', res[i][2])
		}
	}

	regex = regexp.MustCompile(`[^ A-Z](-)`)
	res = regex.FindAllStringSubmatchIndex(episode.AnimeTitle, -1)

	for i := range res {
		episode.AnimeTitle = episode.AnimeTitle[:res[i][2]] + " " + episode.AnimeTitle[res[i][2]:]
	}

	regex = regexp.MustCompile(` ([\d]+)`)
	resMatch := regex.FindAllStringSubmatch(episode.EpisodeNumber, 1)
	if len(resMatch) > 0 {
		episode.EpisodeNumber = strings.TrimSpace(resMatch[0][1])
	}

	episode.AnimeTitle = strings.TrimSpace(episode.AnimeTitle)
	if episode.EpisodeNumber == "." {
		episode.EpisodeNumber = ""
	}

}

func replaceAtIndex(str string, replacement rune, index int) string {
	out := []rune(str)
	out[index] = replacement
	return string(out)
}

func parseEpisodeName(episodeInfo *EpisodeInfo, patterns []FilenamePattern) bool {
	for _, pattern := range patterns {
		if pattern.Pattern.MatchString(episodeInfo.FileName) {
			pattern.Function(episodeInfo)
			break
		}
	}
	if episodeInfo.AnimeTitle == "" {
		log.Printf("Could not parse %s", episodeInfo.FileName)
		return false
	}

	return true
}

func findAllMediaFiles() ([]string, []string) {
	files := findAllFiles()
	var result = []string{}
	var unknown = []string{}
	for _, file := range files {
		ext := strings.ToUpper(filepath.Ext(file))
		if slices.Contains(MEDIA_EXTENSIONS, ext) {
			result = append(result, file)
		} else if !slices.Contains(unknown, ext) {
			unknown = append(unknown, ext)
		}

	}
	return result, unknown
}

func findAllFiles() []string {
	var files = []string{}
	findAllFilesInDir(".", &files)
	return files
}

var MIN_FILE_SIZE int64 = 1024 * 1024 // 1MB

func findAllFilesInDir(dir string, result *[]string) {
	files, _ := os.ReadDir(dir)
	for _, file := range files {
		if file.IsDir() {
			if (file.Name() != ".") && (file.Name() != "..") {
				findAllFilesInDir(dir+"/"+file.Name(), result)
			}
		} else {
			fullPath, _ := filepath.Abs(dir + "/" + file.Name())
			fileInfo, _ := os.Stat(fullPath)
			if fileInfo.Size() > MIN_FILE_SIZE {
				*result = append(*result, fullPath)
			}
		}
	}
}
