package ytdl

import (
	"bufio"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// FormatList is a slice of formats with filtering functionality
type FormatList []Format

func (formats FormatList) Filter(key FormatKey, values []interface{}) FormatList {
	var dst FormatList
	for _, v := range values {
		for _, f := range formats {
			if interfaceToString(f.ValueForKey(key)) == interfaceToString(v) {
				dst = append(dst, f)
			}
		}
	}

	return dst
}

func (formats FormatList) Extremes(key FormatKey, best bool) FormatList {
	dst := formats.Copy()
	if len(dst) > 1 {
		dst.Sort(key, best)
		first := dst[0]
		var i int
		for i = 0; i < len(dst)-1; i++ {
			if first.CompareKey(dst[i+1], key) != 0 {
				break
			}
		}
		i++
		dst = dst[0:i]
	}
	return dst
}

func (formats FormatList) Best(key FormatKey) FormatList {
	return formats.Extremes(key, true)
}

func (formats FormatList) Worst(key FormatKey) FormatList {
	return formats.Extremes(key, false)
}

func (formats FormatList) Sort(key FormatKey, reverse bool) {
	wrapper := formatsSortWrapper{formats, key}
	if !reverse {
		sort.Stable(wrapper)
	} else {
		sort.Stable(sort.Reverse(wrapper))
	}
}

func (formats FormatList) Subtract(other FormatList) FormatList {
	var dst FormatList
	for _, f := range formats {
		include := true
		for _, f2 := range other {
			if f2.Itag == f.Itag {
				include = false
				break
			}
		}
		if include {
			dst = append(dst, f)
		}
	}
	return dst
}

func (formats FormatList) Copy() FormatList {
	dst := make(FormatList, len(formats))
	copy(dst, formats)
	return dst
}

func (formats *FormatList) parseFormats(rd io.Reader) {
	r := bufio.NewReader(rd)

	for {
		line, err := r.ReadString(',')

		if err == io.EOF {
			break
		}

		formats.parseFormat(line[:len(line)-1])
	}
}

func (formats *FormatList) parseFormat(input string) {
	query, err := url.ParseQuery(input)
	if err != nil {
		log.Debug().Msgf("Unable to parse format as query: %v", query)
		return
	}

	itag, _ := strconv.Atoi(query.Get("itag"))
	format, ok := newFormat(itag)
	if !ok {
		log.Debug().Msgf("No metadata found for itag: %v, skipping...", itag)
		return
	}

	if strings.HasPrefix(query.Get("conn"), "rtmp") {
		format.meta["rtmp"] = true
	}
	for k, v := range query {
		if len(v) == 1 {
			format.meta[k] = v[0]
		} else {
			format.meta[k] = v
		}
	}

	*formats = append(*formats, format)
}

type formatsSortWrapper struct {
	formats FormatList
	key     FormatKey
}

func (s formatsSortWrapper) Len() int {
	return len(s.formats)
}

func (s formatsSortWrapper) Less(i, j int) bool {
	return s.formats[i].CompareKey(s.formats[j], s.key) < 0
}

func (s formatsSortWrapper) Swap(i, j int) {
	s.formats[i], s.formats[j] = s.formats[j], s.formats[i]
}
