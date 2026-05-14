package dash0

import (
	"bytes"
	"encoding/json"
)

// UnmarshalJSON tolerates two wire shapes for the generated [Error] type that
// the Dash0 control plane is observed to emit:
//
//	{"code": int, "message": string, "traceId": string}   (canonical struct)
//	"<message>"                                           (bare string)
//
// The bare-string form is lifted into Message; Code stays 0 and TraceId stays
// nil. Marshal always emits the struct form, regardless of how the value was
// decoded; consumers that round-trip an Error through JSON will get the
// canonical shape on the way out.
func (e *Error) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var msg string
		if err := json.Unmarshal(trimmed, &msg); err != nil {
			return err
		}
		*e = Error{Message: msg}
		return nil
	}
	type alias Error
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*e = Error(aux)
	return nil
}
