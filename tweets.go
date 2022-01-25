// -*- tab-width: 4; -*-

package main

import (
	"bufio"
	"encoding/base32"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
)

type TWTMeta struct {
	Nick string
	URLs []string
}

type Tweeter struct {
	Nick string
	URL  string
	Meta TWTMeta
}

type Tweet struct {
	Tweeter Tweeter
	Created time.Time
	Text    string
}

func (tweet *Tweet) Hash() string {
	var authorURL string
	if len(tweet.Tweeter.Meta.URLs) > 0 {
		authorURL = tweet.Tweeter.Meta.URLs[0]
	} else {
		authorURL = tweet.Tweeter.URL
	}

	source := fmt.Sprintf("%s\n%s\n%s", authorURL, tweet.Created.Format(time.RFC3339), tweet.Text)
	sum := blake2b.Sum256([]byte(source))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding)
	hash := strings.ToLower(encoded.EncodeToString(sum[:]))
	hash = hash[len(hash)-7:]

	return hash
}

func (tweet *Tweet) RepliesTo(hash string) bool {
	return strings.HasPrefix(tweet.Text, fmt.Sprintf("(#%s)", hash))
}

// Thread holds all tweets related to a hash (aka conversation or thread)
type Thread struct {
	Root    Tweet
	Replies Tweets
}

// typedef to be able to attach sort methods
type Tweets []Tweet

func (tweets Tweets) Len() int {
	return len(tweets)
}
func (tweets Tweets) Less(i, j int) bool {
	return tweets[i].Created.Before(tweets[j].Created)
}
func (tweets Tweets) Swap(i, j int) {
	tweets[i], tweets[j] = tweets[j], tweets[i]
}

func (tweets Tweets) Tags() map[string]int {
	tags := make(map[string]int)
	re := regexp.MustCompile(`#[-\w]+`)
	for _, tweet := range tweets {
		for _, tag := range re.FindAllString(tweet.Text, -1) {
			tags[strings.TrimLeft(tag, "#")]++
		}
	}
	return tags
}

func (tweets Tweets) Thread(hash string) Thread {
	var thread Thread
	hash = strings.Replace(hash, "#", "", 1)

	if hash == "" {
		return thread
	}

	for _, tweet := range tweets {
		if tweet.Hash() == hash {
			thread.Root = tweet
		} else if tweet.RepliesTo(hash) {
			thread.Replies = append(thread.Replies, tweet)
		}
	}

	return thread
}

func ParseFile(scanner *bufio.Scanner, tweeter Tweeter) Tweets {
	var tweets Tweets
	re := regexp.MustCompile(`^(.+?)(\s+)(.+)$`) // .+? is ungreedy
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			parseTweeterMeta(&tweeter, line)
		}
		parts := re.FindStringSubmatch(line)
		// "Submatch 0 is the match of the entire expression, submatch 1 the
		// match of the first parenthesized subexpression, and so on."
		if len(parts) != 4 {
			if debug {
				log.Printf("could not parse: '%s' (source:%s)\n", line, tweeter.URL)
			}
			continue
		}
		tweets = append(tweets,
			Tweet{
				Tweeter: tweeter,
				Created: ParseTime(parts[1]),
				Text:    parts[3],
			})
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return tweets
}

func ParseTime(timestr string) time.Time {
	var tm time.Time
	var err error
	// Twtxt clients generally uses basically time.RFC3339Nano, but sometimes
	// there's a colon in the timezone, or no timezone at all.
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05.999999999Z0700",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04.999999999Z07:00",
		"2006-01-02T15:04.999999999Z0700",
		"2006-01-02T15:04.999999999",
	} {
		tm, err = time.Parse(layout, strings.ToUpper(timestr))
		if err != nil {
			continue
		} else {
			break
		}
	}
	if err != nil {
		return time.Unix(0, 0)
	}
	return tm
}

// parseTweeterMeta parses twt file metadata and sets them
// for the provided tweeter instance
func parseTweeterMeta(tweeter *Tweeter, line string) {
	re := regexp.MustCompile("^#\\s?(\\w+)\\s?=\\s?(.+)$")
	items := re.FindStringSubmatch(line)

	if len(items) < 3 { // full match, 1st match group, 2nd match group
		return
	}

	meta := items[1]
	value := items[2]

	switch meta {
	case "nick":
		tweeter.Meta.Nick = value
	case "url":
		tweeter.Meta.URLs = append(tweeter.Meta.URLs, value)
	}
}
