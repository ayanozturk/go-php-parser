package analyse

import "testing"

func TestNormalizeTypeWithContextPreservesDNFGrouping(t *testing.T) {
	ctx := FileTypeContext{
		Namespace: "App",
		Aliases: map[string]string{
			"left": "Vendor\\LeftContract",
		},
	}
	got := normalizeTypeWithContext("(Left&Right)|Third", ctx)
	want := "(Vendor\\LeftContract&App\\Right)|App\\Third"
	if got != want {
		t.Fatalf("normalized DNF = %q, want %q", got, want)
	}
	if reparsed := ParseType(got).dnfString(); reparsed != "(App\\Right&Vendor\\LeftContract)|App\\Third" {
		t.Fatalf("reparsed normalized DNF = %q", reparsed)
	}
}

func TestNormalizeTemplateAwareTypePreservesDNFGrouping(t *testing.T) {
	ctx := FileTypeContext{Namespace: "App", Aliases: map[string]string{}}
	got := normalizeTemplateAwareType("(T&Marker)|Alternative", ctx, map[string]struct{}{"T": {}})
	want := "(T&App\\Marker)|App\\Alternative"
	if got != want {
		t.Fatalf("template-aware DNF = %q, want %q", got, want)
	}
}

func TestReturnTypeRuleResolvesUseAliasForNewExpression(t *testing.T) {
	php := `<?php
namespace App;

use Vendor\Package\User as ImportedUser;

function makeUser(): ImportedUser {
    return new ImportedUser();
}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for imported class alias, got: %#v", issues)
	}
}

func TestArgumentTypeRuleResolvesNamespacedPropertyAndAlias(t *testing.T) {
	php := `<?php
namespace App;

use Vendor\Package\User as ImportedUser;

class Example {
    private ImportedUser $user;

    public function takesUser(ImportedUser $user): void {
    }

    public function run(): void {
        $this->takesUser($this->user);
    }
}`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for imported alias property passed to parameter, got: %#v", issues)
	}
}

func TestPropertyTypeRuleResolvesCurrentNamespaceClassType(t *testing.T) {
	php := `<?php
namespace App\Domain;

class User {}

class Example {
    private User $user;

    public function assign(): void {
        $this->user = new User();
    }
}`
	issues := analysePHP(t, php)
	if hasPropertyTypeIssue(issues) {
		t.Fatalf("expected no A.PROP.TYPE issue for current namespace class assignment, got: %#v", issues)
	}
}

func TestReturnTypeRuleAcceptsSubclassForParentReturn(t *testing.T) {
	php := `<?php
namespace App;

class BaseUser {}
class AdminUser extends BaseUser {}

function makeUser(): BaseUser {
    return new AdminUser();
}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for subclass returned as parent type, got: %#v", issues)
	}
}

func TestArgumentTypeRuleAcceptsSubclassForParentParameter(t *testing.T) {
	php := `<?php
namespace App;

class BaseUser {}
class AdminUser extends BaseUser {}

class Example {
    public function takesUser(BaseUser $user): void {
    }

    public function run(): void {
        $admin = new AdminUser();
        $this->takesUser($admin);
    }
}`
	issues := analysePHP(t, php)
	if hasArgTypeIssue(issues) {
		t.Fatalf("expected no A.ARG.TYPE issue for subclass passed to parent parameter, got: %#v", issues)
	}
}
