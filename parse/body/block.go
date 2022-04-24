package body

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var (
	ErrInvalidAttributionFormat = errors.New("invalid attributionFormat")
	ErrInvalidRegex             = errors.New("invalid regex")
)

const (
	attributionNameRegexPart                 = `(?:[^<>,\s]|[^<>,\s][^<>,]*[^<>,\s])`
	attributionEmailRegexPart                = `[^<>@\s]+@[^<>@\s]+`
	attributionGroupEmailRegexPart           = `[^\s@]+@(?:yahoogroups\.com|y?\.{3})`
	attributionShortMonthRegexPart           = `(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)`
	attributionShortWeekdayRegexPart         = `(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun)`
	attributionTimeRegexPart                 = `\d{2}:\d{2}:\d{2}`
	attributionNumericTimezoneRegexPart      = `[-+]\d{4}`
	attributionAbbreviationTimezoneRegexPart = `\([A-Z]{2,}\)`
	nonNewlineWhitespaceRegexPart            = `[\t ]*`
)

const (
	attributionLongDateFormat             = "Mon, 2 Jan 2006"
	attributionShortDateFormat            = "Mon, 01/02/06"
	attributionNumericTimezoneFormat      = "Mon, 2 Jan 2006 15:04:05 -0700"
	attributionAbbreviationTimezoneFormat = "Mon, 2 Jan 2006 15:04:05 -0700 (MST)"
)

type attributionFormat string

const (
	attributionFormatName                         attributionFormat = "Name"
	attributionFormatNameLongDate                 attributionFormat = "NameLongDate"
	attributionFormatNameShortDate                attributionFormat = "NameShortDate"
	attributionFormatNameDateNumericTimezone      attributionFormat = "NameDateNumericTimezone"
	attributionFormatNameDateAbbreviationTimezone attributionFormat = "NameDateAbbreviationTimezone"
)

func (f attributionFormat) HasTime() bool {
	switch f {
	case attributionFormatNameDateNumericTimezone, attributionFormatNameDateAbbreviationTimezone:
		return true
	case attributionFormatName, attributionFormatNameLongDate, attributionFormatNameShortDate:
		return false
	default:
		panic(fmt.Errorf("%w: %s", ErrInvalidAttributionFormat, f))
	}
}

func (f attributionFormat) DateFormat() *string {
	var format string

	switch f {
	case attributionFormatName:
		return nil
	case attributionFormatNameLongDate:
		format = attributionLongDateFormat
	case attributionFormatNameShortDate:
		format = attributionShortDateFormat
	case attributionFormatNameDateNumericTimezone:
		format = attributionNumericTimezoneFormat
	case attributionFormatNameDateAbbreviationTimezone:
		format = attributionAbbreviationTimezoneFormat
	default:
		panic(fmt.Errorf("%w: %s", ErrInvalidAttributionFormat, f))
	}

	return &format
}

type attributionRegex struct {
	Format            attributionFormat
	Regex             *regexp.Regexp
	NameCaptureGroups []int
	TimeCaptureGroups []int
}

func indicesForSubmatch(number int, match []int) []int {
	return []int{match[2*number], match[2*number+1]}
}

func (r attributionRegex) TimeIndices(match []int) []int {
	if r.TimeCaptureGroups == nil {
		return nil
	}

	for _, captureGroup := range r.TimeCaptureGroups {
		// Try each capture group until we find the first one that matched.
		submatchIndices := indicesForSubmatch(captureGroup, match)
		startIndex, endIndex := submatchIndices[0], submatchIndices[1]
		if startIndex >= 0 && endIndex >= 0 {
			return []int{startIndex, endIndex}
		}
	}

	panic(ErrInvalidRegex)
}

func (r attributionRegex) NameIndices(match []int) []int {
	if r.NameCaptureGroups == nil {
		return nil
	}

	for _, captureGroup := range r.NameCaptureGroups {
		// Try each capture group until we find the first one that matched.
		submatchIndices := indicesForSubmatch(captureGroup, match)
		startIndex, endIndex := submatchIndices[0], submatchIndices[1]
		if startIndex >= 0 && endIndex >= 0 {
			return []int{startIndex, endIndex}
		}
	}

	panic(ErrInvalidRegex)
}

var (
	messageHeaderBannerRegexPart               = fmt.Sprintf(`%[1]s-+ ?Original Message ?-+%[1]s`, nonNewlineWhitespaceRegexPart)
	attributionUserCapturingRegexPart          = fmt.Sprintf(`(?:"(%[1]s)"\s+<%[2]s>|(%[1]s)\s+<%[2]s>|<(%[2]s)>|(%[1]s))`, attributionNameRegexPart, attributionEmailRegexPart)
	attributionUserWithEmailCapturingRegexPart = fmt.Sprintf(`(?:"(%[1]s)"\s+<%[2]s>|(%[1]s)\s+<%[2]s>|<(%[2]s)>)`, attributionNameRegexPart, attributionEmailRegexPart)
	longDateRegexPart                          = fmt.Sprintf(`%s, \d{1,2} %s \d{4}`, attributionShortWeekdayRegexPart, attributionShortMonthRegexPart)
	shortDateRegexPart                         = fmt.Sprintf(`%s, \d{2}/\d{2}/\d{2}`, attributionShortWeekdayRegexPart)
	timeWithNumericTimezoneRegexPart           = fmt.Sprintf(`%s %s %s`, longDateRegexPart, attributionTimeRegexPart, attributionNumericTimezoneRegexPart)
	timeWithAbbreviationTimezoneRegexPart      = fmt.Sprintf(`%s %s %s %s`, longDateRegexPart, attributionTimeRegexPart, attributionNumericTimezoneRegexPart, attributionAbbreviationTimezoneRegexPart)
)

var (
	dividerRegex            = regexp.MustCompile(fmt.Sprintf(`(?m)^%[1]s[-_]{2,}%[1]s$`, nonNewlineWhitespaceRegexPart))
	fieldLabelRegex         = regexp.MustCompile(fmt.Sprintf(`(?m)^%s(From|Reply-To|To|Subject|Date|Sent|Message): +(\S)`, nonNewlineWhitespaceRegexPart))
	messageHeaderStartRegex = regexp.MustCompile(fmt.Sprintf(`(?:^%[2]s\n|^%[1]s\n?|\n%[1]s(?:%[2]s)?\n)%[1]s(From|Reply-To|To|Subject|Date|Sent|Message): +(\S)`, nonNewlineWhitespaceRegexPart, messageHeaderBannerRegexPart))
	messageHeaderEndRegex   = regexp.MustCompile(fmt.Sprintf(`(?m)^%s\n`, nonNewlineWhitespaceRegexPart))
)

var attributionRegexes = []attributionRegex{
	{
		Format: attributionFormatNameDateAbbreviationTimezone,
		Regex: regexp.MustCompile(fmt.Sprintf(
			`(?m)^%[1]sOn\s+(%[2]s),\s+%[3]s\s+wrote:%[1]s$`,
			nonNewlineWhitespaceRegexPart,
			timeWithAbbreviationTimezoneRegexPart,
			attributionUserCapturingRegexPart,
		)),
		NameCaptureGroups: []int{2, 3, 4, 5},
		TimeCaptureGroups: []int{1},
	},
	{
		Format: attributionFormatNameDateNumericTimezone,
		Regex: regexp.MustCompile(fmt.Sprintf(
			`(?m)^%[1]sOn\s+(%[2]s),\s+%[3]s\s+wrote:%[1]s$`,
			nonNewlineWhitespaceRegexPart,
			timeWithNumericTimezoneRegexPart,
			attributionUserCapturingRegexPart,
		)),
		NameCaptureGroups: []int{2, 3, 4, 5},
		TimeCaptureGroups: []int{1},
	},
	{
		Format: attributionFormatNameLongDate,
		Regex: regexp.MustCompile(fmt.Sprintf(
			`(?m)^%[1]sOn\s+(%[2]s),\s+%[3]s\s+wrote:%[1]s$`,
			nonNewlineWhitespaceRegexPart,
			longDateRegexPart,
			attributionUserCapturingRegexPart,
		)),
		NameCaptureGroups: []int{2, 3, 4, 5},
		TimeCaptureGroups: []int{1},
	},
	{
		Format: attributionFormatNameShortDate,
		Regex: regexp.MustCompile(fmt.Sprintf(
			`(?m)^%[1]s-{2,3}\s+On\s+(%[2]s),\s+%[3]s\s+wrote:%[1]s$`,
			nonNewlineWhitespaceRegexPart,
			shortDateRegexPart,
			attributionUserCapturingRegexPart,
		)),
		NameCaptureGroups: []int{2, 3, 4, 5},
		TimeCaptureGroups: []int{1},
	},
	{
		Format: attributionFormatName,
		Regex: regexp.MustCompile(fmt.Sprintf(
			`(?m)^%[1]s-{2,3}\s+In\s+%[2]s,\s+%[3]s\s+wrote:%[1]s$`,
			nonNewlineWhitespaceRegexPart,
			attributionGroupEmailRegexPart,
			attributionUserCapturingRegexPart,
		)),
		NameCaptureGroups: []int{1, 2, 3, 4},
	},
	{
		Format: attributionFormatName,
		Regex: regexp.MustCompile(fmt.Sprintf(
			`(?m)^%[1]s-{2,3}\s+%[2]s\s+wrote:%[1]s$`,
			nonNewlineWhitespaceRegexPart,
			attributionUserCapturingRegexPart,
		)),
		NameCaptureGroups: []int{1, 2, 3, 4},
	},
	{
		Format: attributionFormatName,
		Regex: regexp.MustCompile(fmt.Sprintf(
			`(?m)^%[1]s%[2]s%[1]swrote:\s+`,
			nonNewlineWhitespaceRegexPart,
			attributionUserWithEmailCapturingRegexPart,
		)),
		NameCaptureGroups: []int{1, 2, 3},
	},
}

type Block interface {
	ToHtml() string
	FromText(text string) (ok bool, before, after string)
}

type Field struct {
	Name  string
	Value string
}

type MessageHeaderBlock []Field

type messageHeaderFieldPosition struct {
	LabelStartIndex int
	LabelEndIndex   int
	ValueStartIndex int
}

func (b *MessageHeaderBlock) FromText(text string) (ok bool, before, after string) {
	var fieldPositions []messageHeaderFieldPosition

	remaining := text
	currentIndex := 0
	absoluteFieldListEndIndex := len(text)

	if match := messageHeaderStartRegex.FindStringSubmatchIndex(remaining); match != nil {
		position := messageHeaderFieldPosition{
			LabelStartIndex: match[2],
			LabelEndIndex:   match[3],
			ValueStartIndex: match[4],
		}
		fieldPositions = append(fieldPositions, position)

		currentIndex += position.LabelEndIndex
		remaining = remaining[position.LabelEndIndex:]

		matchStartIndex := match[0]
		before = text[:matchStartIndex]
	} else {
		return false, "", ""
	}

	if match := messageHeaderEndRegex.FindStringIndex(remaining); match != nil {
		relativeStartIndex, relativeEndIndex := match[0], match[1]
		absoluteStartIndex, absoluteEndIndex := currentIndex+relativeStartIndex, currentIndex+relativeEndIndex

		remaining = remaining[:relativeStartIndex]
		absoluteFieldListEndIndex = absoluteStartIndex
		after = text[absoluteEndIndex:]
	}

	for {
		match := fieldLabelRegex.FindStringSubmatchIndex(remaining)
		if match == nil {
			break
		}

		relativeFieldStartIndex, relativeFieldEndIndex, relativeValueStartIndex := match[2], match[3], match[4]

		position := messageHeaderFieldPosition{
			LabelStartIndex: currentIndex + relativeFieldStartIndex,
			LabelEndIndex:   currentIndex + relativeFieldEndIndex,
			ValueStartIndex: currentIndex + relativeValueStartIndex,
		}
		fieldPositions = append(fieldPositions, position)

		currentIndex += relativeFieldEndIndex
		remaining = remaining[relativeFieldEndIndex:]
	}

	for i, position := range fieldPositions {
		if i+1 < len(fieldPositions) {
			nextPosition := fieldPositions[i+1]

			*b = append(*b, Field{
				Name:  text[position.LabelStartIndex:position.LabelEndIndex],
				Value: text[position.ValueStartIndex:nextPosition.LabelStartIndex],
			})
		} else {
			*b = append(*b, Field{
				Name:  text[position.LabelStartIndex:position.LabelEndIndex],
				Value: text[position.ValueStartIndex:absoluteFieldListEndIndex],
			})
		}
	}

	return true, before, after
}

type DividerBlock struct{}

func (DividerBlock) FromText(text string) (ok bool, before, after string) {
	match := dividerRegex.FindStringIndex(text)
	if match == nil {
		return false, "", ""
	}

	matchStartIndex, matchEndIndex := match[0], match[1]

	return true, text[:matchStartIndex], text[matchEndIndex:]
}

type AttributionBlock struct {
	Name    string
	Time    *time.Time
	HasTime bool
}

func (b *AttributionBlock) FromText(text string) (ok bool, before, after string) {
	for _, regex := range attributionRegexes {
		match := regex.Regex.FindStringSubmatchIndex(text)
		if match == nil {
			continue
		}

		matchStartIndex, matchEndIndex := match[0], match[1]
		nameIndices := regex.NameIndices(match)

		b.Name = text[nameIndices[0]:nameIndices[1]]

		if dateFormat := regex.Format.DateFormat(); dateFormat != nil {
			timeIndices := regex.TimeIndices(match)
			localTime, err := time.Parse(*dateFormat, text[timeIndices[0]:timeIndices[1]])
			if err != nil {
				continue
			}

			dateTime := localTime.UTC()
			b.Time = &dateTime
		}

		b.HasTime = regex.Format.HasTime()

		return true, text[:matchStartIndex], text[matchEndIndex:]
	}

	return false, "", ""
}
