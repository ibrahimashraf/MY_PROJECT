package manifestreceiptbridge

import (
	"testing"
	"time"
)

func TestLoadAllowsBridgeAbsence(t *testing.T) {
	context, writer, err := Load("", "", "", time.Now)
	if err != nil || context != nil || writer != nil {
		t.Fatalf("absence result = context:%#v writer:%#v err:%v", context, writer, err)
	}
}

func TestLoadRejectsPartialOrUnsupportedContext(t *testing.T) {
	if _, _, err := Load("0123456789abcdef0123456789abcdef", t.TempDir(), "", time.Now); err == nil {
		t.Fatal("partial context was accepted")
	}
	if _, _, err := Load("0123456789abcdef0123456789abcdef", t.TempDir(), "3", time.Now); err == nil {
		t.Fatal("unsupported context was accepted")
	}
}

func TestLoadBuildsValidatedV2Writer(t *testing.T) {
	context, writer, err := Load("0123456789abcdef0123456789abcdef", t.TempDir(), "2", time.Now)
	if err != nil || context == nil || writer == nil {
		t.Fatalf("valid context failed: context:%#v writer:%#v err:%v", context, writer, err)
	}
}
