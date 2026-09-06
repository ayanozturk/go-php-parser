package analyse

import "testing"

func TestGenericCollectionMatchesAfterErasingTypeArguments(t *testing.T) {
	declared := ParseType(`Doctrine\Common\Collections\Collection<string, App\Entity\User>`)
	actual := ParseType(`Doctrine\Common\Collections\Collection`)
	if !declared.Accepts(actual) {
		t.Fatalf("expected Collection<string, User> to accept Collection after erasing type arguments")
	}
}

func TestWithRelativeClassNamesRewritesSelfStaticAndParent(t *testing.T) {
	got := ParseType("self|static|parent|null").withRelativeClassNames(
		`App\Status`,
		`App\OpenStatus`,
		`App\BaseStatus`,
	)
	if got.String() != `App\BaseStatus|App\OpenStatus|App\Status|null` {
		t.Fatalf("rewrote relative class names incorrectly: %q", got.String())
	}
	if ParseType("int").withRelativeClassNames(`App\Status`, `App\Status`, "").String() != "int" {
		t.Fatalf("non-relative types must stay unchanged")
	}
}

func TestObjectTypeAcceptsPhpunitDoubleWithUnboundTemplate(t *testing.T) {
	for _, marker := range []string{
		`PHPUnit\Framework\MockObject\MockObject`,
		`PHPUnit\Framework\MockObject\Stub`,
		`PHPUnit\Framework\MockObject\StubInternal`,
	} {
		t.Run(marker, func(t *testing.T) {
			declared := ParseType(`Example\Domain\Record`)
			actual := ParseType(marker + `&PHPUnit\Framework\RealInstanceType`)

			if !declared.Accepts(actual) {
				t.Fatalf("expected generated test-double intersection to satisfy an object type")
			}
		})
	}
}

func TestScalarTypeRejectsPhpunitMockWithUnboundTemplate(t *testing.T) {
	declared := ParseType("string")
	actual := ParseType(`PHPUnit\Framework\MockObject\MockObject&PHPUnit\Framework\RealInstanceType`)

	if declared.Accepts(actual) {
		t.Fatalf("expected generated mock intersection not to satisfy a scalar type")
	}
}
