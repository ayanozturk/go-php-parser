package analyse

import "testing"

func TestCallableAcceptsClosure(t *testing.T) {
	if !ParseType("callable").Accepts(ParseType("Closure")) {
		t.Fatal("expected callable to accept Closure")
	}
}

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

func TestAsPhpunitMockIntersectionRewritesUnion(t *testing.T) {
	const (
		userClass = `App\User`
		mockClass = `PHPUnit\Framework\MockObject\MockObject`
	)

	assertMockIntersection := func(t *testing.T, got Type, className, mockName string) {
		t.Helper()
		gotDNF := got.dnfString()
		for _, expected := range []string{
			ParseType(className + "&" + mockName).dnfString(),
			ParseType(mockName + "&" + className).dnfString(),
		} {
			if gotDNF == expected {
				return
			}
		}
		t.Fatalf("mock intersection DNF = %q, want %q&%q in either order", gotDNF, className, mockName)
	}

	assertNullableMockIntersection := func(t *testing.T, got Type, className, mockName string) {
		t.Helper()
		gotDNF := got.dnfString()
		for _, intersection := range []string{
			ParseType(className + "&" + mockName).dnfString(),
			ParseType(mockName + "&" + className).dnfString(),
		} {
			for _, expected := range []string{
				ParseType("(" + intersection + ")|null").dnfString(),
				intersection + "|null",
			} {
				if gotDNF == expected {
					return
				}
			}
		}
		t.Fatalf("nullable mock intersection DNF = %q, want (%q&%q)|null in either order", gotDNF, className, mockName)
	}

	assertUnchanged := func(t *testing.T, raw string) {
		t.Helper()
		original := ParseType(raw)
		got := original.asPhpunitMockIntersection()
		if got.dnfString() != original.dnfString() {
			t.Fatalf("ParseType(%q).asPhpunitMockIntersection().dnfString() = %q, want unchanged %q", raw, got.dnfString(), original.dnfString())
		}
	}

	assertMockIntersection(t, ParseType(userClass+"|"+mockClass).asPhpunitMockIntersection(), userClass, mockClass)
	assertNullableMockIntersection(t, ParseType(userClass+"|"+mockClass+"|null").asPhpunitMockIntersection(), userClass, mockClass)
	assertUnchanged(t, "FirstChoice|SecondChoice")
	assertUnchanged(t, userClass+"&"+mockClass)
}

func TestRefineTypeByInstanceofKeepsMockIntersection(t *testing.T) {
	assertRefined := func(t *testing.T, currentRaw, assertedRaw, wantRaw string) {
		t.Helper()
		got := refineTypeByInstanceof(ParseType(currentRaw), ParseType(assertedRaw))
		want := ParseType(wantRaw)
		if got.dnfString() != want.dnfString() {
			t.Fatalf("refineTypeByInstanceof(ParseType(%q), ParseType(%q)).dnfString() = %q, want %q", currentRaw, assertedRaw, got.dnfString(), want.dnfString())
		}
	}

	assertRefined(t,
		`PHPUnit\Framework\MockObject\MockObject&App\TemplateShift|null`,
		`App\TemplateShift`,
		`PHPUnit\Framework\MockObject\MockObject&App\TemplateShift`,
	)
	assertRefined(t, `UserInterface|null`, `User`, `User`)
	assertRefined(t, `App\User|null`, `App\User`, `App\User`)
}
