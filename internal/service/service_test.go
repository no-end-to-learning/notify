package service

import "testing"

func TestValidateChannelRequiresCanonicalName(t *testing.T) {
	for _, name := range []string{"feishu", "telegram"} {
		channel, err := ValidateChannel(name)
		if err != nil {
			t.Fatalf("ValidateChannel(%q) error = %v", name, err)
		}
		if string(channel) != name {
			t.Fatalf("ValidateChannel(%q) = %q", name, channel)
		}
	}

	for _, name := range []string{"Feishu", "Telegram", "FEISHU"} {
		if _, err := ValidateChannel(name); err == nil {
			t.Fatalf("ValidateChannel(%q) error = nil", name)
		}
	}
}
