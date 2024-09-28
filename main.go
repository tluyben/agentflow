package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/dop251/goja"
	"github.com/joho/godotenv"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)


type Flow struct {
	Name          string          `json:"name" yaml:"name"`
	Model         string          `json:"model" yaml:"model"`
	Actions       []string        `json:"action" yaml:"action"`
	Input         json.RawMessage `json:"input" yaml:"input"`
	Output        json.RawMessage `json:"output" yaml:"output"`
	SystemPrompt  string          `json:"system-prompt" yaml:"system-prompt"`
	Prompt        string          `json:"prompt" yaml:"prompt"`
	FlowSteps     []FlowStep      `json:"flow" yaml:"flow"`
}

type FlowStep struct {
	Validate string `json:"validate" yaml:"validate"`
	Next     string `json:"next" yaml:"next"`
}

type Action struct {
	Name   string `json:"name" yaml:"name"`
	Script string `json:"script" yaml:"script"`
}

type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type Document struct {
	ID      string
	Content string
	Vector  []float32
}

var (
	flows     map[string]Flow
	actions   map[string]Action
	envVars   map[string]string
	flowVars  map[string]string
	forceFlag bool
	orKey     string
	orModelHigh string
	orModelLow  string
	searchIndex bleve.Index
)

func main() {
	app := &cli.App{
		Name:  "agentflow",
		Usage: "A tool for managing AI workflows with embedded search",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "flows",
				Usage: "Directory containing flow definitions",
				Value: "flows",
			},
			&cli.StringFlag{
				Name:  "actions",
				Usage: "Directory containing action definitions",
				Value: "actions",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Execute shell commands without asking for confirmation",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "start",
				Usage:  "Start a flow",
				Action: startFlow,
			},
			{
				Name:   "index",
				Usage:  "Index or reindex all files in the current directory and subdirectories",
				Action: indexFiles,
			},
			{
				Name:   "search",
				Usage:  "Search indexed files and start a flow with the results",
				Action: searchAndStartFlow,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	godotenv.Load()
	flows = make(map[string]Flow)
	actions = make(map[string]Action)
	envVars = make(map[string]string)
	flowVars = make(map[string]string)

	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		envVars[pair[0]] = pair[1]
	}

	orKey = os.Getenv("OR_KEY")
	orModelHigh = os.Getenv("OR_MODEL_HIGH")
	orModelLow = os.Getenv("OR_MODEL_LOW")

	if orKey == "" || orModelHigh == "" || orModelLow == "" {
		fmt.Println("Error: OR_KEY, OR_MODEL_HIGH, and OR_MODEL_LOW environment variables must be set")
		os.Exit(1)
	}

	// Open or create the search index
	var err error
	searchIndex, err = openOrCreateIndex("agentflow.bleve")
	if err != nil {
		fmt.Printf("Error opening or creating search index: %v\n", err)
		os.Exit(1)
	}
}

func openOrCreateIndex(indexPath string) (bleve.Index, error) {
	index, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		log.Println("Index doesn't exist. Creating a new one.")
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			return nil, fmt.Errorf("error creating new index: %w", err)
		}
	} else if err != nil {
		log.Printf("Error opening index: %v. Attempting to delete and recreate.\n", err)
		err = deleteIndex(indexPath)
		if err != nil {
			return nil, fmt.Errorf("error deleting corrupted index: %w", err)
		}
		mapping := bleve.NewIndexMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			return nil, fmt.Errorf("error creating new index after deletion: %w", err)
		}
	}
	return index, nil
}

func deleteIndex(indexPath string) error {
	err := os.RemoveAll(indexPath)
	if err != nil {
		return fmt.Errorf("error deleting index directory: %w", err)
	}
	log.Println("Existing index deleted.")
	return nil
}

func isTextFile(path string) bool {

	// check if the file size is > 512 length continue otherwise return false;
	if info, err := os.Stat(path); err == nil {
		if info.Size() < 512 {
			return false
		}
	}
		
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file %s: %v\n", path, err)
		return false
	}
	defer file.Close()

	// Read the first 512 bytes of the file
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		log.Printf("Error reading file %s: %v\n", path, err)
		return false
	}

	// Use the http.DetectContentType function to detect the content type
	contentType := http.DetectContentType(buffer)

	// log the content type and file name 
	// log.Printf("Content Type: %s, File Name: %s\n", contentType, path)
	
	// Check if the content type starts with "text/"
	return strings.HasPrefix(contentType, "text/")
}

func indexFiles(c *cli.Context) error {
	log.Println("Starting indexing process...")

	// Delete existing index before reindexing
	err := deleteIndex("agentflow.bleve")
	if err != nil {
		return fmt.Errorf("error deleting existing index: %w", err)
	}

	// Recreate the index
	searchIndex, err = openOrCreateIndex("agentflow.bleve")
	if err != nil {
		return fmt.Errorf("error creating new index: %w", err)
	}

	batch := searchIndex.NewBatch()
	batchCount := 0
	maxBatchSize := 100

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %q: %v\n", path, err)
			return err
		}

		// Skip .git directory, .bleve directory, and their contents
		if info.IsDir() && (info.Name() == ".git" || info.Name() == "agentflow.bleve" || 
			strings.Contains(path, string(os.PathSeparator)+".git"+string(os.PathSeparator)) ||
			strings.Contains(path, string(os.PathSeparator)+"agentflow.bleve"+string(os.PathSeparator))) {
			return filepath.SkipDir
		}

		if !info.IsDir() && isTextFile(path) {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				log.Printf("Error reading file %s: %v\n", path, err)
				return nil // Continue with next file
			}

			doc := struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			}{
				ID:      path,
				Content: string(content),
			}

			err = batch.Index(doc.ID, doc)
			if err != nil {
				log.Printf("Error adding document %s to batch: %v\n", path, err)
				return nil // Continue with next file
			}

			batchCount++
			log.Printf("Added to batch: %s\n", path)

			if batchCount >= maxBatchSize {
				err = searchIndex.Batch(batch)
				if err != nil {
					log.Printf("Error indexing batch: %v\n", err)
					// Here, we could choose to return the error if it's critical
					// For now, we'll log it and continue
				}
				batch = searchIndex.NewBatch()
				batchCount = 0
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Error walking file path: %v\n", err)
		// Decide whether to return here or continue with indexing the documents we've gathered
	}

	// Index any remaining documents
	if batchCount > 0 {
		err = searchIndex.Batch(batch)
		if err != nil {
			log.Printf("Error indexing final batch: %v\n", err)
		}
	}

	log.Println("Indexing complete.")
	return nil
}


func searchAndStartFlow(c *cli.Context) error {
	if c.NArg() < 2 {
		return fmt.Errorf("please provide a search query and a flow name")
	}

	query := c.Args().Get(0)
	flowName := c.Args().Get(1)

	searchResults, err := search(query)
	if err != nil {
		return fmt.Errorf("error searching: %w", err)
	}

	flow, ok := flows[flowName]
	if !ok {
		return fmt.Errorf("flow %s not found", flowName)
	}

	input := fmt.Sprintf("Search Query: %s\n\nSearch Results:\n%s", query, searchResults)
	return executeFlow(flow, input)
}

func search(query string) (string, error) {
	q := bleve.NewMatchQuery(query)
	searchRequest := bleve.NewSearchRequest(q)
	searchRequest.Fields = []string{"content"}
	searchResults, err := searchIndex.Search(searchRequest)
	if err != nil {
		return "", fmt.Errorf("error performing search: %w", err)
	}

	var results strings.Builder
	for _, hit := range searchResults.Hits {
		results.WriteString(fmt.Sprintf("File: %s\n", hit.ID))
		if content, ok := hit.Fields["content"].(string); ok {
			results.WriteString(fmt.Sprintf("Content: %s\n", content))
		}
		results.WriteString("\n")
	}

	return results.String(), nil
}

func loadFlowsAndActions(c *cli.Context) error {
	flowsDir := c.String("flows")
	actionsDir := c.String("actions")
	forceFlag = c.Bool("force")

	err := loadFlows(flowsDir)
	if err != nil {
		return fmt.Errorf("error loading flows: %w", err)
	}

	err = loadActions(actionsDir)
	if err != nil {
		return fmt.Errorf("error loading actions: %w", err)
	}

	return nil
}

func loadFlows(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".yml")) {
			flow, err := loadFlow(path)
			if err != nil {
				return fmt.Errorf("error loading flow from %s: %w", path, err)
			}
			flows[flow.Name] = flow
		}
		return nil
	})
}

func loadFlow(path string) (Flow, error) {
	var flow Flow
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return flow, fmt.Errorf("error reading file %s: %w", path, err)
	}

	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(data, &flow)
	} else if strings.HasSuffix(path, ".yml") {
		err = yaml.Unmarshal(data, &flow)
	}

	if err != nil {
		return flow, fmt.Errorf("error parsing file %s: %w", path, err)
	}

	return flow, nil
}

func loadActions(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".yml")) {
			action, err := loadAction(path)
			if err != nil {
				return fmt.Errorf("error loading action from %s: %w", path, err)
			}
			actions[action.Name] = action
		}
		return nil
	})
}

func loadAction(path string) (Action, error) {
	var action Action
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return action, fmt.Errorf("error reading file %s: %w", path, err)
	}

	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(data, &action)
	} else if strings.HasSuffix(path, ".yml") {
		err = yaml.Unmarshal(data, &action)
	}

	if err != nil {
		return action, fmt.Errorf("error parsing file %s: %w", path, err)
	}

	return action, nil
}

func startFlow(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("please provide an input for the flow")
	}

	err := loadFlowsAndActions(c)
	if err != nil {
		return err
	}

	userInput := c.Args().First()
	startFlow, ok := flows["start"]
	if !ok {
		return fmt.Errorf("start flow not found")
	}

	return executeFlow(startFlow, userInput)
}

func executeFlow(flow Flow, input string) error {
	var model string
	if flow.Model == "high" {
		model = orModelHigh
	} else if flow.Model == "low" {
		model = orModelLow
	} else {
		return fmt.Errorf("invalid model specified in flow: %s", flow.Model)
	}

	validatedInput, err := validateWithLLM(model, flow.Input, input)
	if err != nil {
		return fmt.Errorf("error validating input: %w", err)
	}

	systemPrompt := substituteVariables(flow.SystemPrompt, envVars, flowVars)
	userPrompt := substituteVariables(flow.Prompt, envVars, flowVars)
	if userPrompt == "" {
		userPrompt = validatedInput
	} else {
		userPrompt = strings.ReplaceAll(userPrompt, "{USER}", validatedInput)
	}

	llmOutput, err := callLLM(model, systemPrompt, userPrompt)
	if err != nil {
		return fmt.Errorf("error calling LLM: %w", err)
	}

	validatedOutput, err := validateWithLLM(model, flow.Output, llmOutput)
	if err != nil {
		return fmt.Errorf("error validating output: %w", err)
	}

	for _, actionName := range flow.Actions {
		action, ok := actions[actionName]
		if !ok {
			return fmt.Errorf("action %s not found", actionName)
		}
		validatedOutput, err = executeAction(action, validatedOutput)
		if err != nil {
			return fmt.Errorf("error executing action %s: %w", actionName, err)
		}
	}

	err = processOutput(validatedOutput)
	if err != nil {
		return fmt.Errorf("error processing output: %w", err)
	}

	for _, step := range flow.FlowSteps {
		valid, err := evaluateJSCondition(step.Validate, map[string]interface{}{
			"input":  validatedInput,
			"output": validatedOutput,
		})
		if err != nil {
			return fmt.Errorf("error evaluating condition: %w", err)
		}
		if valid {
			if step.Next == "$END" {
				return nil
			}
			nextFlow, ok := flows[step.Next]
			if !ok {
				return fmt.Errorf("flow %s not found", step.Next)
			}
			return executeFlow(nextFlow, validatedOutput)
		}
	}

	return fmt.Errorf("no valid next step found")
}

func validateWithLLM(model string, schema json.RawMessage, input string) (string, error) {
	systemPrompt := `You are a JSON validator. Your task is to validate the given input against the provided JSON schema. If the input is valid, return it as is. If it's not valid, modify it to fit the schema. Always return a valid JSON object.`
	userPrompt := fmt.Sprintf("Schema: %s\n\nInput: %s", string(schema), input)

	output, err := callLLM(model, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("error calling LLM for validation: %w", err)
	}

	return output, nil
}

func callLLM(model, systemPrompt, userPrompt string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	requestBody := OpenRouterRequest{
		Model:    model,
		Messages: messages,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %w", err)
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+orKey)
	req.Header.Set("HTTP-Referer", "https://github.com/tluyben/agentflow")
	req.Header.Set("X-Title", "AgentFlow")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to OpenRouter: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenRouter API returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
	}

	var openRouterResp OpenRouterResponse
	err = json.Unmarshal(body, &openRouterResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenRouter")
	}

	return openRouterResp.Choices[0].Message.Content, nil
}



func executeAction(action Action, input string) (string, error) {
	vm := goja.New()
	vm.Set("input", input)

	result, err := vm.RunString(action.Script)
	if err != nil {
		return "", fmt.Errorf("error executing action script: %w", err)
	}

	return result.String(), nil
}

func processOutput(output string) error {
	result := gjson.Parse(output)

	if setVars := result.Get("set-variable").Array(); len(setVars) > 0 {
		for _, v := range setVars {
			if v.IsArray() && len(v.Array()) == 2 {
				flowVars[v.Array()[0].String()] = v.Array()[1].String()
			}
		}
	}

	if removeVars := result.Get("remove-variable").Array(); len(removeVars) > 0 {
		for _, v := range removeVars {
			delete(flowVars, v.String())
		}
	}

	if shellCmd := result.Get("run-shell").String(); shellCmd != "" {
		if !forceFlag {
			fmt.Printf("Do you want to execute the following shell command? (y/n)\n%s\n", shellCmd)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				return nil
			}
		}

		cmd := exec.Command("sh", "-c", shellCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error executing shell command: %w", err)
		}
	}

	return nil
}

func evaluateJSCondition(code string, variables map[string]interface{}) (bool, error) {
	vm := goja.New()
	for k, v := range variables {
		vm.Set(k, v)
	}
	result, err := vm.RunString(code)
	if err != nil {
		return false, fmt.Errorf("error evaluating JS condition: %w", err)
	}
	return result.ToBoolean(), nil
}

func substituteVariables(input string, envVars, flowVars map[string]string) string {
	result := input

	for k, v := range envVars {
		result = strings.ReplaceAll(result, fmt.Sprintf("{{%s}}", k), v)
	}

	for k, v := range flowVars {
		result = strings.ReplaceAll(result, fmt.Sprintf("[[%s]]", k), v)
	}

	return result
}