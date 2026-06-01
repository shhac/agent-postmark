package cli

import "testing"

func TestParseServerSetupSpecs(t *testing.T) {
	specs, err := parseServerSetupSpecs(
		[]string{"app:101:outbound", "billing:102:broadcasts"},
		[]string{"app=server_app", "billing=server_billing"},
		"billing",
	)
	if err != nil {
		t.Fatalf("parseServerSetupSpecs: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("len(specs) = %d", len(specs))
	}
	if specs[0].Alias != "app" || specs[0].ID != 101 || specs[0].Stream != "outbound" || specs[0].Token != "server_app" || specs[0].Default {
		t.Fatalf("specs[0] = %#v", specs[0])
	}
	if specs[1].Alias != "billing" || specs[1].ID != 102 || specs[1].Stream != "broadcasts" || specs[1].Token != "server_billing" || !specs[1].Default {
		t.Fatalf("specs[1] = %#v", specs[1])
	}
}

func TestParseServerSetupSpecsDefaultsFirstServer(t *testing.T) {
	specs, err := parseServerSetupSpecs([]string{"app:101"}, nil, "")
	if err != nil {
		t.Fatalf("parseServerSetupSpecs: %v", err)
	}
	if len(specs) != 1 || specs[0].Stream != "outbound" || !specs[0].Default {
		t.Fatalf("specs = %#v", specs)
	}
}

func TestParseServerSetupSpecsRejectsUnknownDefault(t *testing.T) {
	_, err := parseServerSetupSpecs([]string{"app:101"}, nil, "billing")
	if err == nil {
		t.Fatal("expected error")
	}
}
