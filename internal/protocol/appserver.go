package protocol

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/akatsuki-kk/codex-notifier/internal/notifier"
)

type AppServerEnvelope struct {
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     json.RawMessage `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type commandApprovalParams struct {
	ItemID                 string                 `json:"itemId"`
	ThreadID               string                 `json:"threadId"`
	TurnID                 string                 `json:"turnId"`
	Reason                 string                 `json:"reason"`
	Command                string                 `json:"command"`
	Cwd                    string                 `json:"cwd"`
	NetworkApprovalContext *networkApprovalPrompt `json:"networkApprovalContext"`
}

type networkApprovalPrompt struct {
	Host     string `json:"host"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

type fileApprovalParams struct {
	ItemID    string `json:"itemId"`
	ThreadID  string `json:"threadId"`
	TurnID    string `json:"turnId"`
	Reason    string `json:"reason"`
	GrantRoot string `json:"grantRoot"`
}

type requestUserInputParams struct {
	ThreadID  string              `json:"threadId"`
	TurnID    string              `json:"turnId"`
	RequestID string              `json:"requestId"`
	Questions []inputQuestionItem `json:"questions"`
}

type inputQuestionItem struct {
	Question string                `json:"question"`
	Options  []inputQuestionOption `json:"options"`
}

type inputQuestionOption struct {
	Label string `json:"label"`
}

type turnCompletedParams struct {
	Turn turnPayload `json:"turn"`
}

type turnPayload struct {
	ID       string     `json:"id"`
	Status   string     `json:"status"`
	ThreadID string     `json:"threadId"`
	Error    *turnError `json:"error"`
}

type turnError struct {
	Message string `json:"message"`
}

type threadStatusChangedParams struct {
	ThreadID string       `json:"threadId"`
	Status   threadStatus `json:"status"`
}

type threadStatus struct {
	Type        string   `json:"type"`
	ActiveFlags []string `json:"activeFlags"`
}

func ToNotificationFromAppServer(method string, params json.RawMessage, requestID string) (notifier.Event, bool) {
	switch method {
	case "item/commandExecution/requestApproval":
		var payload commandApprovalParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return notifier.Event{}, false
		}
		return commandApprovalNotification(payload, requestID), true
	case "item/fileChange/requestApproval":
		var payload fileApprovalParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return notifier.Event{}, false
		}
		return fileApprovalNotification(payload, requestID), true
	case "item/tool/requestUserInput", "tool/requestUserInput":
		var payload requestUserInputParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return notifier.Event{}, false
		}
		return requestUserInputNotification(payload, requestID), true
	case "turn/completed":
		var payload turnCompletedParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return notifier.Event{}, false
		}
		return turnCompletedNotification(payload), true
	case "thread/status/changed":
		var payload threadStatusChangedParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return notifier.Event{}, false
		}
		return threadStatusChangedNotification(payload), true
	default:
		return notifier.Event{}, false
	}
}

func commandApprovalNotification(payload commandApprovalParams, requestID string) notifier.Event {
	body := "コマンド実行の確認待ちがあります"
	if payload.NetworkApprovalContext != nil && strings.TrimSpace(payload.NetworkApprovalContext.Host) != "" {
		target := payload.NetworkApprovalContext.Host
		if payload.NetworkApprovalContext.Port != 0 {
			target = fmt.Sprintf("%s:%d", target, payload.NetworkApprovalContext.Port)
		}
		if protocol := strings.TrimSpace(payload.NetworkApprovalContext.Protocol); protocol != "" {
			body = fmt.Sprintf("ネットワーク接続の確認待ちがあります: %s://%s", protocol, target)
		} else {
			body = fmt.Sprintf("ネットワーク接続の確認待ちがあります: %s", target)
		}
	} else if command := strings.TrimSpace(payload.Command); command != "" {
		body = fmt.Sprintf("コマンド実行の確認待ちがあります: %s", truncate(command, 80))
	} else if reason := strings.TrimSpace(payload.Reason); reason != "" {
		body = fmt.Sprintf("コマンド実行の確認待ちがあります: %s", truncate(reason, 80))
	}

	return notifier.Event{
		Category: notifier.CategoryActionRequired,
		Subtitle: "コマンド実行の確認待ち",
		Body:     body,
		Key:      strings.Join([]string{"item/commandExecution/requestApproval", payload.ThreadID, payload.TurnID, payload.ItemID, requestID}, "|"),
	}
}

func fileApprovalNotification(payload fileApprovalParams, requestID string) notifier.Event {
	body := "ファイル変更の確認待ちがあります"
	if root := strings.TrimSpace(payload.GrantRoot); root != "" {
		body = fmt.Sprintf("ファイル変更の確認待ちがあります: %s", truncate(root, 80))
	} else if reason := strings.TrimSpace(payload.Reason); reason != "" {
		body = fmt.Sprintf("ファイル変更の確認待ちがあります: %s", truncate(reason, 80))
	}

	return notifier.Event{
		Category: notifier.CategoryActionRequired,
		Subtitle: "ファイル変更の確認待ち",
		Body:     body,
		Key:      strings.Join([]string{"item/fileChange/requestApproval", payload.ThreadID, payload.TurnID, payload.ItemID, requestID}, "|"),
	}
}

func requestUserInputNotification(payload requestUserInputParams, requestID string) notifier.Event {
	body := "追加の入力待ちがあります"
	if len(payload.Questions) > 0 {
		question := strings.TrimSpace(payload.Questions[0].Question)
		options := len(payload.Questions[0].Options)
		if question != "" && options > 0 {
			body = fmt.Sprintf("追加の入力待ちがあります: %s (%d件の選択肢)", truncate(question, 60), options)
		} else if question != "" {
			body = fmt.Sprintf("追加の入力待ちがあります: %s", truncate(question, 60))
		}
	}

	return notifier.Event{
		Category: notifier.CategoryActionRequired,
		Subtitle: "追加入力の確認待ち",
		Body:     body,
		Key:      strings.Join([]string{"item/tool/requestUserInput", payload.ThreadID, payload.TurnID, payload.RequestID, requestID}, "|"),
	}
}

func turnCompletedNotification(payload turnCompletedParams) notifier.Event {
	body := "Codex の処理が完了しました"
	switch payload.Turn.Status {
	case "failed":
		body = "Codex の処理が失敗しました"
		if payload.Turn.Error != nil && strings.TrimSpace(payload.Turn.Error.Message) != "" {
			body = fmt.Sprintf("%s: %s", body, truncate(payload.Turn.Error.Message, 80))
		}
	case "interrupted":
		body = "Codex の処理が中断されました"
	}

	return notifier.Event{
		Category: notifier.CategoryTurnCompleted,
		Subtitle: "処理完了",
		Body:     body,
		Key:      strings.Join([]string{"turn/completed", payload.Turn.ThreadID, payload.Turn.ID, payload.Turn.Status}, "|"),
	}
}

func threadStatusChangedNotification(payload threadStatusChangedParams) notifier.Event {
	if payload.Status.Type != "active" {
		return notifier.Event{}
	}
	waiting := false
	for _, flag := range payload.Status.ActiveFlags {
		if flag == "waitingOnApproval" {
			waiting = true
			break
		}
	}
	if !waiting {
		return notifier.Event{}
	}

	return notifier.Event{
		Category: notifier.CategoryActionRequired,
		Subtitle: "ユーザー確認待ち",
		Body:     "Codex がユーザー確認を待っています",
		Key:      strings.Join([]string{"thread/status/changed", payload.ThreadID, "waitingOnApproval"}, "|"),
	}
}
