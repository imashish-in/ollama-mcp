package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"
	"google.golang.org/genai"
)

var _ model.ToolCallingChatModel = (*ChatModel)(nil)

// NewChatModel creates a new Gemini chat model instance
//
// Parameters:
//   - ctx: The context for the operation
//   - cfg: Configuration for the Gemini model
//
// Returns:
//   - model.ChatModel: A chat model interface implementation
//   - error: Any error that occurred during creation
//
// Example:
//
//	model, err := gemini.NewChatModel(ctx, &gemini.Config{
//	    Client: client,
//	    Model: "gemini-pro",
//	})
func NewChatModel(_ context.Context, cfg *Config) (*ChatModel, error) {
	return &ChatModel{
		cli:                 cfg.Client,
		model:               cfg.Model,
		maxTokens:           cfg.MaxTokens,
		temperature:         cfg.Temperature,
		topP:                cfg.TopP,
		topK:                cfg.TopK,
		responseSchema:      cfg.ResponseSchema,
		enableCodeExecution: cfg.EnableCodeExecution,
		safetySettings:      cfg.SafetySettings,
	}, nil
}

// Config contains the configuration options for the Gemini model
type Config struct {
	// Client is the Gemini API client instance
	// Required for making API calls to Gemini
	Client *genai.Client

	// Model specifies which Gemini model to use
	// Examples: "gemini-pro", "gemini-pro-vision", "gemini-1.5-flash"
	Model string

	// MaxTokens limits the maximum number of tokens in the response
	// Optional. Example: maxTokens := 100
	MaxTokens *int

	// Temperature controls randomness in responses
	// Range: [0.0, 1.0], where 0.0 is more focused and 1.0 is more creative
	// Optional. Example: temperature := float32(0.7)
	Temperature *float32

	// TopP controls diversity via nucleus sampling
	// Range: [0.0, 1.0], where 1.0 disables nucleus sampling
	// Optional. Example: topP := float32(0.95)
	TopP *float32

	// TopK controls diversity by limiting the top K tokens to sample from
	// Optional. Example: topK := int32(40)
	TopK *int32

	// ResponseSchema defines the structure for JSON responses
	// Optional. Used when you want structured output in JSON format
	ResponseSchema *openapi3.Schema

	// EnableCodeExecution allows the model to execute code
	// Warning: Be cautious with code execution in production
	// Optional. Default: false
	EnableCodeExecution bool

	// SafetySettings configures content filtering for different harm categories
	// Controls the model's filtering behavior for potentially harmful content
	// Optional.
	SafetySettings []*genai.SafetySetting
}

// options contains Gemini-specific options for model configuration
type options struct {
	TopK           *int32
	ResponseSchema *openapi3.Schema
}

type ChatModel struct {
	cli *genai.Client

	model               string
	maxTokens           *int
	topP                *float32
	temperature         *float32
	topK                *int32
	responseSchema      *openapi3.Schema
	tools               []*genai.Tool
	origTools           []*schema.ToolInfo
	toolChoice          *schema.ToolChoice
	enableCodeExecution bool
	safetySettings      []*genai.SafetySetting
}

func (cm *ChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (message *schema.Message, err error) {
	ctx = callbacks.EnsureRunInfo(ctx, cm.GetType(), components.ComponentOfChatModel)

	config, conf, err := cm.buildGenerateConfig(opts...)
	if err != nil {
		return nil, err
	}

	ctx = callbacks.OnStart(ctx, &model.CallbackInput{
		Messages: input,
		Tools:    model.GetCommonOptions(&model.Options{Tools: cm.origTools}, opts...).Tools,
		Config:   conf,
	})
	defer func() {
		if err != nil {
			callbacks.OnError(ctx, err)
		}
	}()

	if len(input) == 0 {
		return nil, fmt.Errorf("gemini input is empty")
	}

	contents, err := cm.convertSchemaMessages(input)
	if err != nil {
		return nil, err
	}

	result, err := cm.cli.Models.GenerateContent(ctx, cm.model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("generate content failed: %w", err)
	}

	message, err = cm.convertResponse(result)
	if err != nil {
		return nil, fmt.Errorf("convert response failed: %w", err)
	}

	callbacks.OnEnd(ctx, cm.convertCallbackOutput(message, conf))
	return message, nil
}

func (cm *ChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (result *schema.StreamReader[*schema.Message], err error) {
	ctx = callbacks.EnsureRunInfo(ctx, cm.GetType(), components.ComponentOfChatModel)

	config, conf, err := cm.buildGenerateConfig(opts...)
	if err != nil {
		return nil, err
	}

	ctx = callbacks.OnStart(ctx, &model.CallbackInput{
		Messages: input,
		Tools:    model.GetCommonOptions(&model.Options{Tools: cm.origTools}, opts...).Tools,
		Config:   conf,
	})
	defer func() {
		if err != nil {
			callbacks.OnError(ctx, err)
		}
	}()

	if len(input) == 0 {
		return nil, fmt.Errorf("gemini input is empty")
	}

	contents, err := cm.convertSchemaMessages(input)
	if err != nil {
		return nil, err
	}

	sr, sw := schema.Pipe[*model.CallbackOutput](1)
	go func() {
		defer func() {
			panicErr := recover()
			if panicErr != nil {
				_ = sw.Send(nil, newPanicErr(panicErr, debug.Stack()))
			}
			sw.Close()
		}()

		for resp, err := range cm.cli.Models.GenerateContentStream(ctx, cm.model, contents, config) {
			if err != nil {
				sw.Send(nil, err)
				return
			}

			message, err := cm.convertResponse(resp)
			if err != nil {
				sw.Send(nil, err)
				return
			}

			closed := sw.Send(cm.convertCallbackOutput(message, conf), nil)
			if closed {
				return
			}
		}
	}()

	srList := sr.Copy(2)
	callbacks.OnEndWithStreamOutput(ctx, srList[0])
	return schema.StreamReaderWithConvert(srList[1], func(t *model.CallbackOutput) (*schema.Message, error) {
		return t.Message, nil
	}), nil
}

func (cm *ChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	if len(tools) == 0 {
		return nil, errors.New("no tools to bind")
	}
	gTools, err := cm.convertToGeminiTools(tools)
	if err != nil {
		return nil, fmt.Errorf("convert to gemini tools failed: %w", err)
	}

	tc := schema.ToolChoiceAllowed
	ncm := *cm
	ncm.toolChoice = &tc
	ncm.tools = gTools
	ncm.origTools = tools
	return &ncm, nil
}

func (cm *ChatModel) BindTools(tools []*schema.ToolInfo) error {
	if len(tools) == 0 {
		return errors.New("no tools to bind")
	}
	gTools, err := cm.convertToGeminiTools(tools)
	if err != nil {
		return err
	}

	cm.tools = gTools
	cm.origTools = tools
	tc := schema.ToolChoiceAllowed
	cm.toolChoice = &tc
	return nil
}

func (cm *ChatModel) BindForcedTools(tools []*schema.ToolInfo) error {
	if len(tools) == 0 {
		return errors.New("no tools to bind")
	}
	gTools, err := cm.convertToGeminiTools(tools)
	if err != nil {
		return err
	}

	cm.tools = gTools
	cm.origTools = tools
	tc := schema.ToolChoiceForced
	cm.toolChoice = &tc
	return nil
}

func (cm *ChatModel) buildGenerateConfig(opts ...model.Option) (*genai.GenerateContentConfig, *model.Config, error) {
	commonOptions := model.GetCommonOptions(&model.Options{
		Temperature: cm.temperature,
		MaxTokens:   cm.maxTokens,
		TopP:        cm.topP,
		Tools:       nil,
		ToolChoice:  cm.toolChoice,
	}, opts...)
	geminiOptions := model.GetImplSpecificOptions(&options{
		TopK:           cm.topK,
		ResponseSchema: cm.responseSchema,
	}, opts...)

	conf := &model.Config{}
	config := &genai.GenerateContentConfig{}

	// Set model
	if commonOptions.Model != nil {
		conf.Model = *commonOptions.Model
	} else {
		conf.Model = cm.model
	}

	// Set temperature
	if commonOptions.Temperature != nil {
		conf.Temperature = *commonOptions.Temperature
		config.Temperature = commonOptions.Temperature
	} else if cm.temperature != nil {
		conf.Temperature = *cm.temperature
		config.Temperature = cm.temperature
	}

	// Set max tokens
	if commonOptions.MaxTokens != nil {
		conf.MaxTokens = *commonOptions.MaxTokens
		config.MaxOutputTokens = int32(*commonOptions.MaxTokens)
	} else if cm.maxTokens != nil {
		conf.MaxTokens = *cm.maxTokens
		config.MaxOutputTokens = int32(*cm.maxTokens)
	}

	// Set top P
	if commonOptions.TopP != nil {
		conf.TopP = *commonOptions.TopP
		config.TopP = commonOptions.TopP
	} else if cm.topP != nil {
		conf.TopP = *cm.topP
		config.TopP = cm.topP
	}

	// Set top K
	if geminiOptions.TopK != nil {
		config.TopK = genai.Ptr(float32(*geminiOptions.TopK))
	} else if cm.topK != nil {
		config.TopK = genai.Ptr(float32(*cm.topK))
	}

	// Set tools
	tools := cm.tools
	if commonOptions.Tools != nil {
		var err error
		tools, err = cm.convertToGeminiTools(commonOptions.Tools)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(tools) > 0 {
		config.Tools = tools
	}

	// Set tool choice
	if commonOptions.ToolChoice != nil {
		switch *commonOptions.ToolChoice {
		case schema.ToolChoiceForbidden:
			config.ToolConfig = &genai.ToolConfig{
				FunctionCallingConfig: &genai.FunctionCallingConfig{
					Mode: genai.FunctionCallingConfigModeNone,
				},
			}
		case schema.ToolChoiceAllowed:
			config.ToolConfig = &genai.ToolConfig{
				FunctionCallingConfig: &genai.FunctionCallingConfig{
					Mode: genai.FunctionCallingConfigModeAuto,
				},
			}
		case schema.ToolChoiceForced:
			if len(tools) == 0 {
				return nil, nil, fmt.Errorf("tool choice is forced but no tools provided")
			}
			config.ToolConfig = &genai.ToolConfig{
				FunctionCallingConfig: &genai.FunctionCallingConfig{
					Mode: genai.FunctionCallingConfigModeAny,
				},
			}
		default:
			return nil, nil, fmt.Errorf("tool choice=%s not supported", *commonOptions.ToolChoice)
		}
	}

	// Set safety settings
	if len(cm.safetySettings) > 0 {
		config.SafetySettings = cm.safetySettings
	}

	// Set response schema for JSON mode
	if geminiOptions.ResponseSchema != nil {
		gSchema, err := cm.convertOpenAPISchema(geminiOptions.ResponseSchema)
		if err != nil {
			return nil, nil, fmt.Errorf("convert response schema failed: %w", err)
		}
		config.ResponseMIMEType = "application/json"
		config.ResponseSchema = gSchema
	}

	return config, conf, nil
}

func (cm *ChatModel) convertToGeminiTools(tools []*schema.ToolInfo) ([]*genai.Tool, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	var functionDeclarations []*genai.FunctionDeclaration
	for _, tool := range tools {
		openSchema, err := tool.ToOpenAPIV3()
		if err != nil {
			return nil, fmt.Errorf("get open schema failed: %w", err)
		}

		gSchema, err := cm.convertOpenAPISchema(openSchema)
		if err != nil {
			return nil, fmt.Errorf("convert open schema failed: %w", err)
		}

		funcDecl := &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Desc,
			Parameters:  gSchema,
		}
		functionDeclarations = append(functionDeclarations, funcDecl)
	}

	return []*genai.Tool{{FunctionDeclarations: functionDeclarations}}, nil
}

func (cm *ChatModel) convertOpenAPISchema(schema *openapi3.Schema) (*genai.Schema, error) {
	if schema == nil {
		return nil, nil
	}

	result := &genai.Schema{
		Description: schema.Description,
	}

	switch schema.Type {
	case openapi3.TypeObject:
		result.Type = genai.TypeObject
		if schema.Properties != nil {
			properties := make(map[string]*genai.Schema)
			for name, prop := range schema.Properties {
				if prop == nil || prop.Value == nil {
					continue
				}
				propSchema, err := cm.convertOpenAPISchema(prop.Value)
				if err != nil {
					return nil, err
				}
				properties[name] = propSchema
			}
			result.Properties = properties
		}
		if schema.Required != nil {
			result.Required = schema.Required
		}
	case openapi3.TypeArray:
		result.Type = genai.TypeArray
		if schema.Items != nil && schema.Items.Value != nil {
			itemSchema, err := cm.convertOpenAPISchema(schema.Items.Value)
			if err != nil {
				return nil, err
			}
			result.Items = itemSchema
		}
	case openapi3.TypeString:
		result.Type = genai.TypeString
		if schema.Enum != nil {
			enums := make([]string, 0, len(schema.Enum))
			for _, e := range schema.Enum {
				if str, ok := e.(string); ok {
					enums = append(enums, str)
				} else {
					return nil, fmt.Errorf("enum value must be a string, schema: %+v", schema)
				}
			}
			result.Enum = enums
		}
	case openapi3.TypeNumber:
		result.Type = genai.TypeNumber
	case openapi3.TypeInteger:
		result.Type = genai.TypeInteger
	case openapi3.TypeBoolean:
		result.Type = genai.TypeBoolean
	default:
		result.Type = genai.TypeUnspecified
	}

	return result, nil
}

func (cm *ChatModel) convertSchemaMessages(messages []*schema.Message) ([]*genai.Content, error) {
	var contents []*genai.Content
	for _, message := range messages {
		content, err := cm.convertSchemaMessage(message)
		if err != nil {
			return nil, fmt.Errorf("convert schema message failed: %w", err)
		}
		if content != nil {
			contents = append(contents, content)
		}
	}
	return contents, nil
}

func (cm *ChatModel) convertSchemaMessage(message *schema.Message) (*genai.Content, error) {
	if message == nil {
		return nil, nil
	}

	var parts []*genai.Part

	// Handle tool calls
	if message.ToolCalls != nil {
		for _, call := range message.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
				return nil, fmt.Errorf("unmarshal tool call arguments failed: %w", err)
			}
			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: call.Function.Name,
					Args: args,
				},
			})
		}
	}

	// Handle tool responses
	if message.Role == schema.Tool {
		var response map[string]any
		if err := json.Unmarshal([]byte(message.Content), &response); err != nil {
			// If the content is not valid JSON, treat it as a plain text error response
			response = map[string]any{
				"error": message.Content,
			}
		}
		parts = append(parts, &genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name:     message.ToolCallID,
				Response: response,
			},
		})
	} else {
		// Handle text content
		if message.Content != "" {
			parts = append(parts, &genai.Part{Text: message.Content})
		}

		// Handle multi-content (images, audio, etc.)
		for _, content := range message.MultiContent {
			switch content.Type {
			case schema.ChatMessagePartTypeText:
				parts = append(parts, &genai.Part{Text: content.Text})
			case schema.ChatMessagePartTypeImageURL:
				if content.ImageURL != nil {
					parts = append(parts, &genai.Part{
						FileData: &genai.FileData{
							MIMEType: content.ImageURL.MIMEType,
							FileURI:  content.ImageURL.URI,
						},
					})
				}
			case schema.ChatMessagePartTypeAudioURL:
				if content.AudioURL != nil {
					parts = append(parts, &genai.Part{
						FileData: &genai.FileData{
							MIMEType: content.AudioURL.MIMEType,
							FileURI:  content.AudioURL.URI,
						},
					})
				}
			case schema.ChatMessagePartTypeVideoURL:
				if content.VideoURL != nil {
					parts = append(parts, &genai.Part{
						FileData: &genai.FileData{
							MIMEType: content.VideoURL.MIMEType,
							FileURI:  content.VideoURL.URI,
						},
					})
				}
			case schema.ChatMessagePartTypeFileURL:
				if content.FileURL != nil {
					parts = append(parts, &genai.Part{
						FileData: &genai.FileData{
							MIMEType: content.FileURL.MIMEType,
							FileURI:  content.FileURL.URI,
						},
					})
				}
			}
		}
	}

	if len(parts) == 0 {
		return nil, nil
	}

	return &genai.Content{
		Role:  string(cm.convertRole(message.Role)),
		Parts: parts,
	}, nil
}

func (cm *ChatModel) convertRole(role schema.RoleType) genai.Role {
	switch role {
	case schema.Assistant:
		return genai.RoleModel
	case schema.User:
		return genai.RoleUser
	case schema.Tool:
		return genai.RoleUser // Tool responses are treated as user messages in the new API
	default:
		return genai.RoleUser
	}
}

func (cm *ChatModel) convertResponse(resp *genai.GenerateContentResponse) (*schema.Message, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini result is empty")
	}

	candidate := resp.Candidates[0]
	message := &schema.Message{
		Role: schema.Assistant,
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: string(candidate.FinishReason),
		},
	}

	// Handle usage metadata
	if resp.UsageMetadata != nil {
		message.ResponseMeta.Usage = &schema.TokenUsage{
			PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
		}
	}

	// Process content parts
	var textParts []string
	for _, part := range candidate.Content.Parts {
		switch {
		case part.Text != "":
			textParts = append(textParts, part.Text)
		case part.FunctionCall != nil:
			args, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				return nil, fmt.Errorf("marshal function call arguments failed: %w", err)
			}
			message.ToolCalls = append(message.ToolCalls, schema.ToolCall{
				ID: part.FunctionCall.Name,
				Function: schema.FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				},
			})
		case part.ExecutableCode != nil:
			textParts = append(textParts, part.ExecutableCode.Code)
		case part.CodeExecutionResult != nil:
			textParts = append(textParts, part.CodeExecutionResult.Output)
		}
	}

	// Set content
	if len(textParts) == 1 {
		message.Content = textParts[0]
	} else if len(textParts) > 1 {
		for _, text := range textParts {
			message.MultiContent = append(message.MultiContent, schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeText,
				Text: text,
			})
		}
	}

	return message, nil
}

func (cm *ChatModel) convertCallbackOutput(message *schema.Message, conf *model.Config) *model.CallbackOutput {
	callbackOutput := &model.CallbackOutput{
		Message: message,
		Config:  conf,
	}
	if message.ResponseMeta != nil && message.ResponseMeta.Usage != nil {
		callbackOutput.TokenUsage = &model.TokenUsage{
			PromptTokens:     message.ResponseMeta.Usage.PromptTokens,
			CompletionTokens: message.ResponseMeta.Usage.CompletionTokens,
			TotalTokens:      message.ResponseMeta.Usage.TotalTokens,
		}
	}
	return callbackOutput
}

func (cm *ChatModel) IsCallbacksEnabled() bool {
	return true
}

const typ = "Gemini"

func (cm *ChatModel) GetType() string {
	return typ
}

type panicErr struct {
	info  any
	stack []byte
}

func (p *panicErr) Error() string {
	return fmt.Sprintf("panic error: %v, \nstack: %s", p.info, string(p.stack))
}

func newPanicErr(info any, stack []byte) error {
	return &panicErr{
		info:  info,
		stack: stack,
	}
}
