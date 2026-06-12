package security

import "testing"

func TestArchiveExtractionRejectsUnsafePaths(t *testing.T) {
	for _, name := range []string{"../evil", "/tmp/evil", "safe/../../evil"} {
		if err := ValidateArchivePath(name); err == nil {
			t.Fatalf("ValidateArchivePath(%q) returned nil", name)
		}
	}
	if err := ValidateArchivePath("safe/file.txt"); err != nil {
		t.Fatalf("safe path rejected: %v", err)
	}
}

func TestExternalCommandRejectsShellMetacharacters(t *testing.T) {
	if err := ValidateCommandArgs([]string{"git", "clone", "owner/repo; rm -rf /"}); err == nil {
		t.Fatalf("unsafe command accepted")
	}
	if err := ValidateCommandArgs([]string{"git", "clone", "owner/repo"}); err != nil {
		t.Fatalf("safe command rejected: %v", err)
	}
}

func TestAPIKeyAndShareableRedaction(t *testing.T) {
	out := Redact("key=secret local=/tmp/private.pdf reviewer=Ada note=private")
	for _, bad := range []string{"secret", "/tmp/private.pdf", "Ada", "private"} {
		if Contains(out, bad) {
			t.Fatalf("redaction leaked %q in %q", bad, out)
		}
	}
}

func TestDataRetentionPolicyAndToolDigestLock(t *testing.T) {
	policy := RetentionPolicy{KeepPrivateNotes: false, RetentionDays: 30}
	if err := policy.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	lock := ExternalToolLock{Name: "grobid", Version: "0.8.0", ContainerDigest: "sha256:abc"}
	if err := lock.Validate(); err != nil {
		t.Fatalf("lock Validate returned error: %v", err)
	}
}

func TestBoundedResponseSize(t *testing.T) {
	if err := CheckResponseSize(1024, 2048); err != nil {
		t.Fatalf("small response rejected: %v", err)
	}
	if err := CheckResponseSize(4096, 2048); err == nil {
		t.Fatalf("large response accepted")
	}
}
