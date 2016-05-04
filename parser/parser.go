package parser

import (
	"bytes"
	"fmt"
	goparser "go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

const (
	SWAGGER_ROUTE_TOKEN = "swagger:route"
	CONSUME_TOKEN       = "consumes:"
	FORM_PARAM_TOKEN    = "@formparam"
	QUERY_PARAM_TOKEN   = "@queryparam"
	PATH_PARAM_TOKEN    = "@pathparam"
	HEADER_PARAM_TOKEN  = "@headerparam"
	BODY_PARAM_TOKEN    = "@bodyparam"

	SWAGGER_FORM_PARAM_TOKEN   = "formData"
	SWAGGER_QUERY_PARAM_TOKEN  = "query"
	SWAGGER_PATH_PARAM_TOKEN   = "path"
	SWAGGER_HEADER_PARAM_TOKEN = "header"
	SWAGGER_BODY_PARAM_TOKEN   = "body"

	STRUCT_FILE_NAME = "structs.go"
)

type Parser struct {
	ResourceList []*Resource
	rPtr         int
	PackageName  string
}

type Resource struct {
	Route            string
	OutputStructName string
	Method           string
	Parameters       map[string]Parameter
	Consumes         []string
}

type Parameter struct {
	Name          string
	ParameterType string
	DataType      string
	IsMandatory   bool
	Description   string
}

func NewParser(pkgNam string) Parser {
	r := make([]*Resource, 0)
	return Parser{ResourceList: r, rPtr: -1, PackageName: pkgNam}
}

func (parser *Parser) ParseRequestParametersIntoStruct(apiFile string) {

	fileSet := token.NewFileSet()
	fileTree, err := goparser.ParseFile(fileSet, apiFile, nil, goparser.ParseComments)
	if err != nil {
		log.Fatalf("Can not parse general API information: %v\n", err)
	}

	if fileTree.Comments != nil {

		var routeFoundFlag bool = false
		for _, comment := range fileTree.Comments {
			if strings.Contains(comment.Text(), SWAGGER_ROUTE_TOKEN) {

				//Swagger route specification
				var consumeFoundFlag bool = false
				for _, commentLine := range strings.Split(comment.Text(), "\n") {
					if routeFoundFlag && consumeFoundFlag {
						c := strings.Split(strings.Trim(commentLine, " "), " ")
						if len(c) < 2 {
							break
						}
						if strings.EqualFold(c[0], "-") {
							parser.ResourceList[parser.rPtr].Consumes = append(parser.ResourceList[parser.rPtr].Consumes, c[1])
							continue
						} else {
							break
						}
					}
					consumeFoundFlag = false
					tmpStr := strings.ToLower(commentLine)
					if strings.Contains(tmpStr, SWAGGER_ROUTE_TOKEN) {
						items := strings.Split(commentLine, " ")
						if len(items) < 4 {
							fmt.Println(commentLine, "- Format is incorrect")
							continue
						}
						routeFoundFlag = true
						rec := &Resource{}
						rec.Method = items[1]
						rec.Route = items[2]
						rec.OutputStructName = items[3]
						params := make(map[string]Parameter)
						rec.Parameters = params
						c := make([]string, 0)
						rec.Consumes = c
						parser.ResourceList = append(parser.ResourceList, rec)
						parser.rPtr = parser.rPtr + 1
					} else if strings.Contains(tmpStr, CONSUME_TOKEN) {
						consumeFoundFlag = true
						continue
					}

				}
				continue

			} else if routeFoundFlag {
				//No route handler, skipping current comment group
				for _, commentLine := range strings.Split(comment.Text(), "\n") {

					commentLine = strings.Trim(commentLine, " ")
					if commentLine == "" {
						//empty comment line
						continue
					}

					si := strings.Index(commentLine, "\"")
					ei := strings.LastIndex(commentLine, "\"")

					if si == 0 || ei <= si {
						continue
					}
					description := commentLine[si+1 : ei]
					commentLine = commentLine[0 : si-1]
					attributes := strings.Split(commentLine, " ")

					if len(attributes) != 4 {
						fmt.Println(commentLine, "- Format incorrect (number of tokens is not 4)")
						continue
					}

					var required bool
					if attributes[3] == "true" {
						required = true
					} else if attributes[3] == "false" {
						required = false
					} else {
						fmt.Println("Invalid require value -", attributes[3], " , in comment line- ", commentLine)
						continue
					}

					//fmt.Println(attribute)
					var qType string
					switch attributes[0] {
					case FORM_PARAM_TOKEN:
						qType = SWAGGER_FORM_PARAM_TOKEN
					case QUERY_PARAM_TOKEN:
						qType = SWAGGER_QUERY_PARAM_TOKEN
					case PATH_PARAM_TOKEN:
						qType = SWAGGER_PATH_PARAM_TOKEN
					case HEADER_PARAM_TOKEN:
						qType = SWAGGER_HEADER_PARAM_TOKEN
					case BODY_PARAM_TOKEN:
						qType = SWAGGER_BODY_PARAM_TOKEN
					default:

						//fmt.Println("Invalid parameter annotaton - ", attributes[0], " ,in line -", commentLine)
						continue
					}

					param := Parameter{Name: attributes[1], ParameterType: qType, DataType: attributes[2], IsMandatory: required, Description: description}
					parser.ResourceList[parser.rPtr].Parameters[attributes[1]] = param
				}

			} else {
				//skipping pre comments as it is
				//no route specification found yet
			}

		}
	}
}

func (parser *Parser) GenerateStructFile() error {
	fd, err := os.Create(parser.PackageName + "/" + STRUCT_FILE_NAME)
	if err != nil {
		return fmt.Errorf("Can not create document file: %v\n", err)
	}
	defer fd.Close()

	var resourceDetails bytes.Buffer
	pNam := parser.PackageName[strings.LastIndex(parser.PackageName, "/")+1:]
	resourceDetails.WriteString("package " + pNam + "\n\n\n")

	for i := 0; i < parser.rPtr+1; i++ {

		if len((*parser.ResourceList[i]).Parameters) <= 0 {
			continue
		}
		resourceDetails.WriteString("//" + (*parser.ResourceList[i]).OutputStructName + " represent the struct for rquest parameters for " + (*parser.ResourceList[i]).Route + " endpoint\n")
		resourceDetails.WriteString("// swagger:parameters " + (*parser.ResourceList[i]).OutputStructName + "\n")
		resourceDetails.WriteString("type " + (*parser.ResourceList[i]).OutputStructName + " struct {\n\n")
		for _, param := range (*parser.ResourceList[i]).Parameters {
			resourceDetails.WriteString("\t//" + param.Description + "\n")
			if param.IsMandatory {
				resourceDetails.WriteString("\t// required: true\n")
			}
			resourceDetails.WriteString("\t// in: " + param.ParameterType + "\n")
			resourceDetails.WriteString("\t" + strings.Title(param.Name) + "\t" + param.DataType + "\t`json:\"" + param.Name + "\"`\n\n")
		}
		resourceDetails.WriteString("}\n\n")
	}

	doc := resourceDetails.String()
	fd.WriteString(doc)

	return nil
}

func (parser *Parser) DeleteStructFile() error {
	return os.Remove(parser.PackageName + "/" + STRUCT_FILE_NAME)
}
