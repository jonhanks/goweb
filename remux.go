package goweb

// from git.ligo.org/jonathan-hanks/cds_metadata_server/
// gpl 3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"regexp/syntax"
)

type ReMuxParamsKey struct{}
type ReMuxMethodsKey struct{}

var reMuxParamsKey ReMuxParamsKey
var reMuxMethodsKey ReMuxMethodsKey

type KeyValPair struct {
	Key interface{}
	Val interface{}
}

type HandlerInfo struct {
	regexp  *syntax.Regexp
	handler http.HandlerFunc
	methods []string
	ctxSeed []KeyValPair
	name    string
}

func (hi *HandlerInfo) Name(name string) *HandlerInfo {
	hi.name = name
	return hi
}

func (hi *HandlerInfo) Methods(methods ...string) *HandlerInfo {
	hi.methods = methods
	return hi
}

func (hi *HandlerInfo) Values(pairs ...KeyValPair) *HandlerInfo {
	hi.ctxSeed = pairs
	return hi
}

type ReMux struct {
	lookup map[*regexp.Regexp]*HandlerInfo
}

func ReMuxParams(r *http.Request) map[string]string {
	if mapping, ok := r.Context().Value(reMuxParamsKey).(map[string]string); ok {
		return mapping
	} else {
		return map[string]string{}
	}
}

func NewReMux() *ReMux {
	return &ReMux{lookup: make(map[*regexp.Regexp]*HandlerInfo)}
}

func (r *ReMux) HandleFunc(path string, handler http.HandlerFunc) *HandlerInfo {
	reTree, err := syntax.Parse(path, syntax.Perl)
	if err != nil {
		panic("Could not compile handler path: " + err.Error())
	}
	if err = validateTree(reTree); err != nil {
		panic("Invalid regexp for the handler path: " + err.Error())
	}

	re := regexp.MustCompile("^" + path + "$")
	fmt.Println(re.String())
	hi := &HandlerInfo{regexp: reTree, handler: handler, methods: make([]string, 0, 0), ctxSeed: make([]KeyValPair, 0, 0)}
	r.lookup[regexp.MustCompile("^"+path+"$")] = hi
	return hi
}

func (r *ReMux) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	matchExceptMethod := false
	for reKey, handlerInfo := range r.lookup {
		// https://stackoverflow.com/questions/20750843/using-named-matches-from-go-regex/20751656
		matches := reKey.FindStringSubmatch(request.URL.Path)
		if matches != nil {
			matchExceptMethod = true

			if len(handlerInfo.methods) > 0 {
				matched := false
				for _, method := range handlerInfo.methods {
					if method == request.Method {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}

			mapping := make(map[string]string)
			for i, name := range reKey.SubexpNames() {
				if name == "" {
					continue
				}
				mapping[name] = matches[i]
			}
			newRequest := request.WithContext(context.WithValue(request.Context(), reMuxParamsKey, mapping))
			for _, ctxPair := range handlerInfo.ctxSeed {
				newRequest = newRequest.WithContext(context.WithValue(newRequest.Context(), ctxPair.Key, ctxPair.Val))
			}
			handlerInfo.handler(writer, newRequest)
			return
		}
	}
	if matchExceptMethod {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = writer.Write([]byte("Unsupported method"))
		return
	}
	http.NotFound(writer, request)
}

func (r *ReMux) Reverse(name string, params map[string]string) (string, error) {
	for _, handlerInfo := range r.lookup {
		if handlerInfo.name == name {
			return expandTree(handlerInfo.regexp, params)
		}
	}
	return "", errors.New("route not found")
}

func expandNode(buf *bytes.Buffer, tree *syntax.Regexp, params map[string]string) error {
	if tree == nil {
		return nil
	}

	switch tree.Op {
	case syntax.OpLiteral:
		for _, r := range tree.Rune {
			_, _ = buf.WriteRune(r)
		}
	case syntax.OpCapture:
		if val, ok := params[tree.Name]; ok {
			_, _ = buf.WriteString(val)
		} else {
			return errors.New("missing parameter " + tree.Name)
		}
	case syntax.OpConcat:
		for _, child := range tree.Sub {
			if err := expandNode(buf, child, params); err != nil {
				return err
			}
		}
	default:
		return errors.New("unexpected node type in regexp")
	}
	return nil
}
func expandTree(tree *syntax.Regexp, params map[string]string) (string, error) {
	buf := &bytes.Buffer{}
	if err := expandNode(buf, tree, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func validateTree(tree *syntax.Regexp) error {
	if tree == nil {
		return nil
	}
	switch tree.Op {
	case syntax.OpLiteral:
		break
	case syntax.OpConcat:
		for _, child := range tree.Sub {
			if err := validateTree(child); err != nil {
				return err
			}
		}
	case syntax.OpCapture:
		if tree.Name == "" {
			return errors.New("top level groupings must be named")
		}
		break
	}
	return nil
}
