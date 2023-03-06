package chatgpt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"github.com/yqchilde/wxbot/engine/pkg/log"
	"github.com/yqchilde/wxbot/engine/robot"
)

var (
	gptClient *openai.Client
	gptModel  *GptModel
)

func AskChatGpt(messages []openai.ChatCompletionMessage, delay ...time.Duration) (answer string, err error) {
	// 获取客户端
	if gptClient == nil {
		gptClient, err = getGptClient()
		if err != nil {
			return "", err
		}
	}

	// 获取模型
	if gptModel == nil {
		gptModel, err = getGptModel()
		if err != nil {
			return "", err
		}
	}

	// 延迟请求
	if len(delay) > 0 {
		time.Sleep(delay[0])
	}

	chatMessages := []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: fmt.Sprintf("你是一个强大的助手，你是ChatGPT，我将为你起一个名字叫%s，并且你会用中文回答我的问题", robot.GetBot().GetConfig().BotNickname),
		},
	}
	chatMessages = append(chatMessages, messages...)

	resp, err := gptClient.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:    gptModel.Model,
		Messages: chatMessages,
	})
	// 处理响应回来的错误
	if err != nil {
		if strings.Contains(err.Error(), "You exceeded your current quota") {
			log.Printf("当前apiKey[%s]配额已用完, 将删除并切换到下一个", apiKeys[0].Key)
			db.Orm.Table("apikey").Where("key = ?", apiKeys[0].Key).Delete(&ApiKey{})
			if len(apiKeys) == 1 {
				return "", errors.New("OpenAi配额已用完，请联系管理员")
			}
			apiKeys = apiKeys[1:]
			gptClient = openai.NewClient(apiKeys[0].Key)
			return AskChatGpt(messages)
		}
		if strings.Contains(err.Error(), "The server had an error while processing your request") {
			log.Println("OpenAi服务出现问题，将重试")
			return AskChatGpt(messages)
		}
		if strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
			log.Println("OpenAi服务请求超时，将重试")
			return AskChatGpt(messages)
		}
		if strings.Contains(err.Error(), "Please reduce your prompt") {
			return "", errors.New("OpenAi免费上下文长度限制为4097个词组，您的上下文长度已超出限制，请发送\"清空会话\"以清空上下文")
		}
		if strings.Contains(err.Error(), "Incorrect API key") {
			return "", errors.New("OpenAi ApiKey错误，请联系管理员")
		}
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// filterAnswer 过滤答案，处理一些符号问题
// return 新的答案，是否需要重试
func filterAnswer(answer string) (newAnswer string, isNeedRetry bool) {
	punctuation := ",，!！?？"
	answer = strings.TrimSpace(answer)
	if len(answer) == 1 {
		return answer, true
	}
	if len(answer) == 3 && strings.ContainsAny(answer, punctuation) {
		return answer, true
	}
	answer = strings.TrimLeftFunc(answer, func(r rune) bool {
		if strings.ContainsAny(string(r), punctuation) {
			return true
		}
		return false
	})
	return answer, false
}

func AskChatGptWithImage(prompt string, delay ...time.Duration) (b64 string, err error) {
	// 获取客户端
	if gptClient == nil {
		gptClient, err = getGptClient()
		if err != nil {
			return "", err
		}
	}

	// 获取模型
	if gptModel == nil {
		gptModel, err = getGptModel()
		if err != nil {
			return "", err
		}
	}

	// 延迟请求
	if len(delay) > 0 {
		time.Sleep(delay[0])
	}

	resp, err := gptClient.CreateImage(context.Background(), openai.ImageRequest{
		Prompt:         prompt,
		Size:           gptModel.ImageSize,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
	})
	if err != nil {
		return "", err
	}
	return resp.Data[0].B64JSON, nil
}
