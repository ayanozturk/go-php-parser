# Style rules

[Rule documentation index](README.md) · [List rules from the CLI](../../README.md#listing-all-style-rules)

These are the 16 style rule codes currently registered by the tool. They can be enabled or disabled with the `rules:` setting. The rule names identify the convention family; actual coverage is limited to the checks described here.

## `Generic.Arrays.DisallowLongArraySyntax`

**What it checks:** Flags `array(...)` and asks for PHP short array syntax `[...]`.

**Why it helps:** One array form across the codebase is easier to scan and keeps new code consistent.

**Example that reports:**

```php
<?php
$items = array(1, 2);
```

**A consistent form:**

```php
<?php
$items = [1, 2];
```

## `Generic.Formatting.DisallowMultipleStatements`

**What it checks:** Flags multiple executable statements written on one line.

**Why it helps:** One statement per line makes control flow, diffs, and review easier to follow.

**Example that reports:**

```php
<?php
$first = 1; $second = 2;
```

**A consistent form:**

```php
<?php
$first = 1;
$second = 2;
```

## `Generic.Functions.FunctionCallArgumentSpacing`

**What it checks:** Checks spacing between arguments in supported function and method calls.

**Why it helps:** Consistent separators make call signatures easier to read and reduce visual clutter.

**Example that reports:**

```php
<?php
store($name,$value);
```

**A consistent form:**

```php
<?php
store($name, $value);
```

## `PSR1.Classes.ClassConstantName`

**What it checks:** Requires class constants to use uppercase letters, digits, and underscores in the supported naming form.

**Why it helps:** A predictable constant style distinguishes fixed class values from properties and methods.

**Example that reports:**

```php
<?php
class Limits { public const maxItems = 10; }
```

**A consistent form:**

```php
<?php
class Limits { public const MAX_ITEMS = 10; }
```

## `PSR1.Classes.ClassDeclaration.PascalCase`

**What it checks:** Requires class-like declaration names to use PascalCase.

**Why it helps:** Consistent type names make declarations easier to find and recognize.

**Example that reports:**

```php
<?php
class user_service {}
```

**A consistent form:**

```php
<?php
class UserService {}
```

## `PSR1.Classes.ClassInstantiation`

**What it checks:** Requires parentheses when instantiating a class, including when there are no constructor arguments.

**Why it helps:** The explicit call form is consistent and leaves a clear place for constructor arguments.

**Example that reports:**

```php
<?php
$service = new Service;
```

**A consistent form:**

```php
<?php
$service = new Service();
```

## `PSR1.Methods.CamelCapsMethodName`

**What it checks:** Requires method names to use camelCase.

**Why it helps:** Consistent method names are easier to predict when reading and calling APIs.

**Example that reports:**

```php
<?php
class User { public function GetName(): string { return "Ada"; } }
```

**A consistent form:**

```php
<?php
class User { public function getName(): string { return "Ada"; } }
```

## `PSR12.Classes.ClosingBraceOnOwnLine`

**What it checks:** Requires class and method closing braces to occupy their own line, with no adjacent code on that line.

**Why it helps:** Braces aligned on separate lines make declaration boundaries easy to scan.

**Example that reports:**

```php
<?php
class User { }
```

**A consistent form:**

```php
<?php
class User
{
}

```

## `PSR12.Classes.OpenBraceOnOwnLine`

**What it checks:** Requires the opening brace of a class, interface, trait, or enum to start its own line.

**Why it helps:** A consistent declaration layout makes class boundaries clear and matches the selected PSR-12 convention.

**Example that reports:**

```php
<?php
class User {
}
```

**A consistent form:**

```php
<?php
class User
{
}
```

## `PSR12.ControlStructures.ControlStructureSpacing`

**What it checks:** Checks selected spacing around control keywords, function-call parentheses, and opening braces.

**Why it helps:** Consistent spacing helps separate control syntax from its condition and body.

**Example that reports:**

```php
<?php
if($ready){run();}
```

**A consistent form:**

```php
<?php
if ($ready) { run(); }
```

## `PSR12.ControlStructures.ElseIfDeclaration`

**What it checks:** Requires the combined `elseif` keyword rather than `else if`.

**Why it helps:** Using the standard spelling avoids inconsistent control-flow formatting.

**Example that reports:**

```php
<?php
if ($a) { one(); } else if ($b) { two(); }
```

**A consistent form:**

```php
<?php
if ($a) { one(); } elseif ($b) { two(); }
```

## `PSR12.Files.EndFileNewline`

**What it checks:** Requires the file to end with exactly one newline.

**Why it helps:** A final newline keeps command-line tools and version-control diffs consistent.

**Example that reports:**

```php
<?php
echo "done";
```

**A consistent form:**

```php
<?php
echo "done";

```

## `PSR12.Files.EndFileNoTrailingWhitespace`

**What it checks:** Flags spaces or tabs at the ends of lines.

**Why it helps:** Trailing whitespace creates noisy diffs and can be invisible during review.

**Example that reports:** The visible `·` markers below each represent a trailing space.

```text
$value = 1;···
```

**A consistent form:**

```php
<?php
$value = 1;
```

## `PSR12.Files.NoBlankLineAfterPHPOpeningTag`

**What it checks:** Requires a blank line immediately after the opening `<?php` tag (the diagnostic message is “Missing blank line…”).

**Why it helps:** The configured convention gives files a consistent visual start after the PHP tag.

**Example that reports:**

```php
<?php
$ready = true;
```

**A consistent form:**

```php
<?php

$ready = true;
```

## `PSR12.Files.NoSpaceBeforeSemicolon`

**What it checks:** Flags a space or tab immediately before a statement-ending semicolon.

**Why it helps:** Removing this stray space keeps statement endings uniform.

**Example that reports:**

```php
<?php
$ready = true ;
```

**A consistent form:**

```php
<?php
$ready = true;
```

## `PSR12.Methods.VisibilityDeclared`

**What it checks:** Requires class methods to declare `public`, `protected`, or `private` visibility.

**Why it helps:** Explicit visibility makes the API surface clear to callers and maintainers.

**Example that reports:**

```php
<?php
class User { function name(): string { return "Ada"; } }
```

**A consistent form:**

```php
<?php
class User { public function name(): string { return "Ada"; } }
```
