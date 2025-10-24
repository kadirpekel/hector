package llms

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
)

func ExtractTextFromMessage(msg *pb.Message) string {
	return protocol.ExtractTextFromMessage(msg)
}

func ExtractToolCallsFromMessage(msg *pb.Message) []*protocol.ToolCall {
	return protocol.GetToolCallsFromMessage(msg)
}

func ExtractToolCallIDFromMessage(msg *pb.Message) string {
	return msg.MessageId
}
