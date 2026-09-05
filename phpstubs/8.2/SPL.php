<?php

/**
 * @template TKey
 * @template TValue
 * @extends Iterator<TKey, TValue>
 */
interface SeekableIterator extends Iterator
{
    public function seek(int $offset): void;
}

/**
 * @template TKey
 * @template TValue
 */
interface ArrayAccess
{
    public function offsetExists(mixed $offset): bool;

    public function offsetGet(mixed $offset): mixed;

    public function offsetSet(mixed $offset, mixed $value): void;

    public function offsetUnset(mixed $offset): void;
}

interface Countable
{
    public function count(): int;
}

interface Serializable
{
    public function serialize(): string;

    public function unserialize(string $data): void;
}

/**
 * @template TKey
 * @template TValue
 * @implements SeekableIterator<TKey, TValue>
 * @implements ArrayAccess<TKey, TValue>
 */
class ArrayIterator implements SeekableIterator, ArrayAccess, Serializable, Countable
{
    public const STD_PROP_LIST = 1;
    public const ARRAY_AS_PROPS = 2;

    public function __construct(array|object $array = [], int $flags = 0) {}

    public function append(mixed $value): void {}

    public function asort(int $flags = 0): bool { return true; }

    public function count(): int { return 0; }

    public function current(): mixed { return null; }

    public function getArrayCopy(): array { return []; }

    public function getFlags(): int { return 0; }

    public function key(): mixed { return null; }

    public function ksort(int $flags = 0): bool { return true; }

    public function natcasesort(): bool { return true; }

    public function natsort(): bool { return true; }

    public function next(): void {}

    public function offsetExists(mixed $key): bool { return true; }

    public function offsetGet(mixed $key): mixed { return null; }

    public function offsetSet(mixed $key, mixed $value): void {}

    public function offsetUnset(mixed $key): void {}

    public function rewind(): void {}

    public function seek(int $offset): void {}

    public function serialize(): string { return ""; }

    public function setFlags(int $flags): void {}

    public function uasort(callable $callback): bool { return true; }

    public function uksort(callable $callback): bool { return true; }

    public function unserialize(string $data): void {}

    public function valid(): bool { return true; }
}

/**
 * @template TKey
 * @template TValue
 * @implements IteratorAggregate<TKey, TValue>
 * @implements ArrayAccess<TKey, TValue>
 */
class ArrayObject implements IteratorAggregate, ArrayAccess, Serializable, Countable
{
    public const STD_PROP_LIST = 1;
    public const ARRAY_AS_PROPS = 2;

    public function __construct(array|object $array = [], int $flags = 0, string $iteratorClass = ArrayIterator::class) {}

    public function append(mixed $value): void {}

    public function asort(int $flags = 0): bool { return true; }

    public function count(): int { return 0; }

    public function exchangeArray(array|object $array): array { return []; }

    public function getArrayCopy(): array { return []; }

    public function getFlags(): int { return 0; }

    public function getIterator(): Iterator { return new ArrayIterator([]); }

    public function getIteratorClass(): string { return ArrayIterator::class; }

    public function ksort(int $flags = 0): bool { return true; }

    public function natcasesort(): bool { return true; }

    public function natsort(): bool { return true; }

    public function offsetExists(mixed $key): bool { return false; }

    public function offsetGet(mixed $key): mixed { return null; }

    public function offsetSet(mixed $key, mixed $value): void {}

    public function offsetUnset(mixed $key): void {}

    public function serialize(): string { return ""; }

    public function setFlags(int $flags): void {}

    public function setIteratorClass(string $iteratorClass): void {}

    public function uasort(callable $callback): bool { return true; }

    public function uksort(callable $callback): bool { return true; }

    public function unserialize(string $data): void {}
}

/**
 * @template TKey
 * @template TValue
 */
class RecursiveArrayIterator extends ArrayIterator implements RecursiveIterator
{
    public const CHILD_ARRAYS_ONLY = 4;

    public function getChildren(): ?RecursiveArrayIterator { return null; }

    public function hasChildren(): bool { return false; }
}

interface OuterIterator extends Iterator
{
    public function getInnerIterator(): ?Iterator;
}

interface RecursiveIterator extends Iterator
{
    public function hasChildren(): bool;

    public function getChildren(): ?RecursiveIterator;
}

class EmptyIterator implements Iterator
{
    public function current(): mixed { return null; }

    public function next(): void {}

    public function key(): mixed { return null; }

    public function valid(): bool { return false; }

    public function rewind(): void {}
}

class IteratorIterator implements OuterIterator
{
    public function __construct(Traversable $iterator, ?string $class = null) {}

    public function getInnerIterator(): ?Iterator { return null; }

    public function current(): mixed { return null; }

    public function key(): mixed { return null; }

    public function next(): void {}

    public function rewind(): void {}

    public function valid(): bool { return true; }
}

class FilterIterator extends IteratorIterator
{
    public function accept(): bool { return true; }
}

class CallbackFilterIterator extends FilterIterator
{
    public function __construct(Iterator $iterator, callable $callback) {}
}

class RecursiveIteratorIterator implements OuterIterator
{
    public const LEAVES_ONLY = 0;
    public const SELF_FIRST = 1;
    public const CHILD_FIRST = 2;
    public const CATCH_GET_CHILD = 16;

    public function __construct(Traversable $iterator, int $mode = self::LEAVES_ONLY, int $flags = 0) {}

    public function getInnerIterator(): ?Iterator { return null; }

    public function getSubIterator(?int $level = null): ?RecursiveIterator { return null; }

    public function getDepth(): int { return 0; }

    public function beginIteration(): void {}

    public function endIteration(): void {}

    public function callHasChildren(): bool { return false; }

    public function callGetChildren(): ?RecursiveIterator { return null; }

    public function beginChildren(): void {}

    public function endChildren(): void {}

    public function nextElement(): void {}

    public function current(): mixed { return null; }

    public function key(): mixed { return null; }

    public function next(): void {}

    public function rewind(): void {}

    public function valid(): bool { return true; }
}

class RecursiveFilterIterator extends FilterIterator implements RecursiveIterator
{
    public function hasChildren(): bool { return false; }

    public function getChildren(): ?RecursiveIterator { return null; }
}

class RecursiveCallbackFilterIterator extends RecursiveFilterIterator
{
    public function __construct(RecursiveIterator $iterator, callable $callback) {}
}

class ParentIterator extends RecursiveFilterIterator
{
    public function accept(): bool { return true; }
}

class LimitIterator extends IteratorIterator
{
    public function __construct(Iterator $iterator, int $offset = 0, int $limit = -1) {}

    public function getPosition(): int { return 0; }

    public function seek(int $offset): void {}
}

class CachingIterator extends IteratorIterator implements ArrayAccess, Countable
{
    public const CALL_TOSTRING = 1;
    public const CATCH_GET_CHILD = 16;
    public const TOSTRING_USE_KEY = 2;
    public const TOSTRING_USE_CURRENT = 4;
    public const TOSTRING_USE_INNER = 8;
    public const FULL_CACHE = 256;

    public function __construct(Iterator $iterator, int $flags = self::CALL_TOSTRING) {}

    public function getCache(): array { return []; }

    public function hasNext(): bool { return false; }

    public function offsetExists(mixed $key): bool { return false; }

    public function offsetGet(mixed $key): mixed { return null; }

    public function offsetSet(mixed $key, mixed $value): void {}

    public function offsetUnset(mixed $key): void {}

    public function count(): int { return 0; }

    public function __toString(): string { return ""; }
}

class RecursiveCachingIterator extends CachingIterator implements RecursiveIterator
{
    public function hasChildren(): bool { return false; }

    public function getChildren(): ?RecursiveIterator { return null; }
}

class NoRewindIterator extends IteratorIterator
{
    public function rewind(): void {}
}

class InfiniteIterator extends IteratorIterator
{
    public function next(): void {}
}

class AppendIterator extends IteratorIterator
{
    public function __construct() {}

    public function append(Iterator $iterator): void {}

    public function getArrayIterator(): ArrayIterator { return new ArrayIterator(); }
}

class MultipleIterator implements Iterator
{
    public const MIT_NEED_ANY = 0;
    public const MIT_NEED_ALL = 1;
    public const MIT_KEYS_NUMERIC = 0;
    public const MIT_KEYS_ASSOC = 2;

    public function __construct(int $flags = self::MIT_NEED_ALL | self::MIT_KEYS_NUMERIC) {}

    public function attachIterator(Iterator $iterator, string|int|null $info = null): void {}

    public function detachIterator(Iterator $iterator): void {}

    public function containsIterator(Iterator $iterator): bool { return false; }

    public function countIterators(): int { return 0; }

    public function current(): array { return []; }

    public function key(): array { return []; }

    public function next(): void {}

    public function rewind(): void {}

    public function valid(): bool { return true; }
}

class SplFileInfo implements Stringable
{
    public function __construct(string $filename) {}

    public function getPath(): string { return ""; }

    public function getFilename(): string { return ""; }

    public function getExtension(): string { return ""; }

    public function getBasename(string $suffix = ""): string { return ""; }

    public function getPathname(): string { return ""; }

    public function getPerms(): int|false { return 0; }

    public function getInode(): int|false { return 0; }

    public function getSize(): int|false { return 0; }

    public function getOwner(): int|false { return 0; }

    public function getGroup(): int|false { return 0; }

    public function getATime(): int|false { return 0; }

    public function getMTime(): int|false { return 0; }

    public function getCTime(): int|false { return 0; }

    public function getType(): string|false { return ""; }

    public function isWritable(): bool { return false; }

    public function isReadable(): bool { return false; }

    public function isExecutable(): bool { return false; }

    public function isFile(): bool { return false; }

    public function isDir(): bool { return false; }

    public function isLink(): bool { return false; }

    public function getLinkTarget(): string|false { return ""; }

    public function getRealPath(): string|false { return ""; }

    public function getFileInfo(?string $class = null): SplFileInfo { return new SplFileInfo(""); }

    public function getPathInfo(?string $class = null): ?SplFileInfo { return null; }

    public function openFile(string $mode = "r", bool $useIncludePath = false, ?resource $context = null): SplFileObject { return new SplFileObject(""); }

    public function setFileClass(string $class = SplFileObject::class): void {}

    public function setInfoClass(string $class = SplFileInfo::class): void {}

    public function __toString(): string { return ""; }
}

class DirectoryIterator extends SplFileInfo implements SeekableIterator
{
    public function __construct(string $directory) {}

    public function current(): mixed { return null; }

    public function key(): mixed { return null; }

    public function next(): void {}

    public function rewind(): void {}

    public function seek(int $offset): void {}

    public function valid(): bool { return true; }

    public function isDot(): bool { return false; }
}

class FilesystemIterator extends DirectoryIterator
{
    public const CURRENT_MODE_MASK = 240;
    public const CURRENT_AS_PATHNAME = 32;
    public const CURRENT_AS_FILEINFO = 0;
    public const CURRENT_AS_SELF = 16;
    public const KEY_MODE_MASK = 3840;
    public const KEY_AS_PATHNAME = 0;
    public const FOLLOW_SYMLINKS = 512;
    public const KEY_AS_FILENAME = 256;
    public const NEW_CURRENT_AND_KEY = 256;
    public const SKIP_DOTS = 4096;
    public const UNIX_PATHS = 8192;

    public function __construct(string $directory, int $flags = self::KEY_AS_PATHNAME | self::CURRENT_AS_FILEINFO | self::SKIP_DOTS) {}

    public function getFlags(): int { return 0; }

    public function setFlags(int $flags): void {}
}

class RecursiveDirectoryIterator extends FilesystemIterator implements RecursiveIterator
{
    public function hasChildren(bool $allowLinks = false): bool { return false; }

    public function getChildren(): ?RecursiveIterator { return null; }

    public function getSubPath(): string { return ""; }

    public function getSubPathname(): string { return ""; }
}

class GlobIterator extends FilesystemIterator implements Countable
{
    public function __construct(string $pattern, int $flags = FilesystemIterator::KEY_AS_PATHNAME | FilesystemIterator::CURRENT_AS_FILEINFO) {}

    public function count(): int { return 0; }
}

class SplFileObject extends SplFileInfo implements RecursiveIterator, SeekableIterator
{
    public const DROP_NEW_LINE = 1;
    public const READ_AHEAD = 2;
    public const SKIP_EMPTY = 4;
    public const READ_CSV = 8;

    public function __construct(string $filename, string $mode = "r", bool $useIncludePath = false, ?resource $context = null) {}

    public function current(): string|array|false { return ""; }

    public function eof(): bool { return false; }

    public function fflush(): bool { return true; }

    public function fgetc(): string|false { return ""; }

    public function fgetcsv(string $separator = ",", string $enclosure = "\"", string $escape = "\\"): array|false { return []; }

    public function fgets(): string { return ""; }

    public function fgetss(string $allowableTags = ""): string|false { return ""; }

    public function flock(int $operation, int &$wouldBlock = null): bool { return true; }

    public function fpassthru(): int { return 0; }

    public function fputcsv(array $fields, string $separator = ",", string $enclosure = "\"", string $escape = "\\", string $eol = "\n"): int|false { return 0; }

    public function fread(int $length): string|false { return ""; }

    public function fscanf(string $format, mixed &...$vars): array|int|null { return []; }

    public function fseek(int $offset, int $whence = SEEK_SET): int { return 0; }

    public function fstat(): array { return []; }

    public function ftell(): int|false { return 0; }

    public function ftruncate(int $size): bool { return true; }

    public function fwrite(string $data, int $length = 0): int|false { return 0; }

    public function getChildren(): ?RecursiveIterator { return null; }

    public function getCsvControl(): array { return []; }

    public function getFlags(): int { return 0; }

    public function getMaxLineLen(): int { return 0; }

    public function hasChildren(): bool { return false; }

    public function key(): int { return 0; }

    public function next(): void {}

    public function rewind(): void {}

    public function seek(int $line): void {}

    public function setCsvControl(string $separator = ",", string $enclosure = "\"", string $escape = "\\"): void {}

    public function setFlags(int $flags): void {}

    public function setMaxLineLen(int $maxLength): void {}

    public function valid(): bool { return true; }
}

class SplTempFileObject extends SplFileObject
{
    public function __construct(int $maxMemory = 2097152) {}
}

class SplDoublyLinkedList implements Iterator, Countable, ArrayAccess, Serializable
{
    public const IT_MODE_LIFO = 2;
    public const IT_MODE_FIFO = 0;
    public const IT_MODE_DELETE = 1;
    public const IT_MODE_KEEP = 0;

    public function add(int $index, mixed $value): void {}

    public function bottom(): mixed { return null; }

    public function count(): int { return 0; }

    public function current(): mixed { return null; }

    public function getIteratorMode(): int { return 0; }

    public function isEmpty(): bool { return true; }

    public function key(): int { return 0; }

    public function next(): void {}

    public function offsetExists(mixed $index): bool { return false; }

    public function offsetGet(mixed $index): mixed { return null; }

    public function offsetSet(mixed $index, mixed $value): void {}

    public function offsetUnset(mixed $index): void {}

    public function pop(): mixed { return null; }

    public function prev(): void {}

    public function push(mixed $value): void {}

    public function rewind(): void {}

    public function serialize(): string { return ""; }

    public function setIteratorMode(int $mode): int { return 0; }

    public function shift(): mixed { return null; }

    public function top(): mixed { return null; }

    public function unserialize(string $data): void {}

    public function unshift(mixed $value): void {}

    public function valid(): bool { return true; }
}

class SplQueue extends SplDoublyLinkedList
{
    public function dequeue(): mixed { return null; }

    public function enqueue(mixed $value): void {}
}

class SplStack extends SplDoublyLinkedList
{
}

class SplHeap implements Iterator, Countable
{
    public function compare(mixed $value1, mixed $value2): int { return 0; }

    public function count(): int { return 0; }

    public function current(): mixed { return null; }

    public function extract(): mixed { return null; }

    public function insert(mixed $value): bool { return true; }

    public function isCorrupted(): bool { return false; }

    public function isEmpty(): bool { return true; }

    public function key(): int { return 0; }

    public function next(): void {}

    public function recoverFromCorruption(): bool { return true; }

    public function rewind(): void {}

    public function top(): mixed { return null; }

    public function valid(): bool { return true; }
}

class SplMaxHeap extends SplHeap
{
}

class SplMinHeap extends SplHeap
{
}

class SplPriorityQueue implements Iterator, Countable
{
    public const EXTR_DATA = 1;
    public const EXTR_PRIORITY = 2;
    public const EXTR_BOTH = 3;

    public function compare(mixed $priority1, mixed $priority2): int { return 0; }

    public function count(): int { return 0; }

    public function current(): mixed { return null; }

    public function extract(): mixed { return null; }

    public function getExtractFlags(): int { return 0; }

    public function insert(mixed $value, mixed $priority): bool { return true; }

    public function isCorrupted(): bool { return false; }

    public function isEmpty(): bool { return true; }

    public function key(): mixed { return null; }

    public function next(): void {}

    public function recoverFromCorruption(): bool { return true; }

    public function rewind(): void {}

    public function setExtractFlags(int $flags): int { return 0; }

    public function top(): mixed { return null; }

    public function valid(): bool { return true; }
}

class SplFixedArray implements IteratorAggregate, ArrayAccess, Countable
{
    public function __construct(int $size = 0) {}

    public function count(): int { return 0; }

    public static function fromArray(array $array, bool $preserveKeys = true): SplFixedArray { return new SplFixedArray(); }

    public function getIterator(): Iterator { return new ArrayIterator([]); }

    public function getSize(): int { return 0; }

    public function offsetExists(mixed $index): bool { return false; }

    public function offsetGet(mixed $index): mixed { return null; }

    public function offsetSet(mixed $index, mixed $value): void {}

    public function offsetUnset(mixed $index): void {}

    public function setSize(int $size): bool { return true; }

    public function toArray(): array { return []; }
}

class SplObjectStorage implements Countable, Iterator, Serializable, ArrayAccess
{
    public function addAll(SplObjectStorage $storage): int { return 0; }

    public function attach(object $object, mixed $info = null): void {}

    public function contains(object $object): bool { return false; }

    public function count(): int { return 0; }

    public function current(): object { return new stdClass(); }

    public function detach(object $object): void {}

    public function getHash(object $object): string { return ""; }

    public function getInfo(): mixed { return null; }

    public function key(): int { return 0; }

    public function next(): void {}

    public function offsetExists(mixed $object): bool { return false; }

    public function offsetGet(mixed $object): mixed { return null; }

    public function offsetSet(mixed $object, mixed $info = null): void {}

    public function offsetUnset(mixed $object): void {}

    public function removeAll(SplObjectStorage $storage): int { return 0; }

    public function removeAllExcept(SplObjectStorage $storage): int { return 0; }

    public function rewind(): void {}

    public function serialize(): string { return ""; }

    public function setInfo(mixed $info): void {}

    public function unserialize(string $data): void {}

    public function valid(): bool { return true; }
}

interface SplObserver
{
    public function update(SplSubject $subject): void;
}

interface SplSubject
{
    public function attach(SplObserver $observer): void;

    public function detach(SplObserver $observer): void;

    public function notify(): void;
}

class LogicException extends Exception {}

class BadFunctionCallException extends LogicException {}

class BadMethodCallException extends BadFunctionCallException {}

class DomainException extends LogicException {}

class InvalidArgumentException extends LogicException {}

class LengthException extends LogicException {}

class OutOfRangeException extends LogicException {}

class RuntimeException extends Exception {}

class OutOfBoundsException extends RuntimeException {}

class OverflowException extends RuntimeException {}

class RangeException extends RuntimeException {}

class UnderflowException extends RuntimeException {}

class UnexpectedValueException extends RuntimeException {}

interface DateTimeInterface extends Stringable
{
    public const ATOM = 'Y-m-d\TH:i:sP';
    public const COOKIE = 'l, d-M-Y H:i:s T';
    public const ISO8601 = 'Y-m-d\TH:i:sO';
    public const ISO8601_EXPANDED = 'X-m-d\TH:i:sP';
    public const RFC822 = 'D, d M y H:i:s O';
    public const RFC850 = 'l, d-M-y H:i:s T';
    public const RFC1036 = 'D, d M y H:i:s O';
    public const RFC1123 = 'D, d M Y H:i:s O';
    public const RFC7231 = 'D, d M Y H:i:s \G\M\T';
    public const RFC2822 = 'D, d M Y H:i:s O';
    public const RFC3339 = 'Y-m-d\TH:i:sP';
    public const RFC3339_EXTENDED = 'Y-m-d\TH:i:s.vP';
    public const RSS = 'D, d M Y H:i:s O';
    public const W3C = 'Y-m-d\TH:i:sP';

    public function diff(DateTimeInterface $targetObject, bool $absolute = false): DateInterval;

    public function format(string $format): string;

    public function getOffset(): int;

    public function getTimestamp(): int;

    public function getTimezone(): DateTimeZone|false;

    public function __toString(): string;
}

class DateTime implements DateTimeInterface
{
    public function __construct(string $datetime = 'now', ?DateTimeZone $timezone = null) {}

    public static function createFromFormat(string $format, string $datetime, ?DateTimeZone $timezone = null): static|false { return new static(); }

    public static function createFromImmutable(DateTimeImmutable $object): static { return new static(); }

    public static function createFromInterface(DateTimeInterface $object): static { return new static(); }

    public static function getLastErrors(): array|false { return false; }

    public function add(DateInterval $interval): static { return $this; }

    public function diff(DateTimeInterface $targetObject, bool $absolute = false): DateInterval { return new DateInterval('P0D'); }

    public function format(string $format): string { return ''; }

    public function getOffset(): int { return 0; }

    public function getTimestamp(): int { return 0; }

    public function getTimezone(): DateTimeZone|false { return false; }

    public function modify(string $modifier): DateTime { return $this; }

    public function setDate(int $year, int $month, int $day): static { return $this; }

    public function setISODate(int $year, int $week, int $dayOfWeek = 1): static { return $this; }

    public function setTime(int $hour, int $minute, int $second = 0, int $microsecond = 0): static { return $this; }

    public function setTimestamp(int $timestamp): static { return $this; }

    public function setTimezone(DateTimeZone $timezone): static { return $this; }

    public function sub(DateInterval $interval): static { return $this; }

    public function __toString(): string { return ''; }
}

class DateTimeImmutable implements DateTimeInterface
{
    public function __construct(string $datetime = 'now', ?DateTimeZone $timezone = null) {}

    public static function createFromFormat(string $format, string $datetime, ?DateTimeZone $timezone = null): static|false { return new static(); }

    public static function createFromMutable(DateTime $object): static { return new static(); }

    public static function createFromInterface(DateTimeInterface $object): static { return new static(); }

    public static function getLastErrors(): array|false { return false; }

    public function add(DateInterval $interval): static { return $this; }

    public function diff(DateTimeInterface $targetObject, bool $absolute = false): DateInterval { return new DateInterval('P0D'); }

    public function format(string $format): string { return ''; }

    public function getOffset(): int { return 0; }

    public function getTimestamp(): int { return 0; }

    public function getTimezone(): DateTimeZone|false { return false; }

    public function modify(string $modifier): DateTimeImmutable { return $this; }

    public function setDate(int $year, int $month, int $day): static { return $this; }

    public function setISODate(int $year, int $week, int $dayOfWeek = 1): static { return $this; }

    public function setTime(int $hour, int $minute, int $second = 0, int $microsecond = 0): static { return $this; }

    public function setTimestamp(int $timestamp): static { return $this; }

    public function setTimezone(DateTimeZone $timezone): static { return $this; }

    public function sub(DateInterval $interval): static { return $this; }

    public function __toString(): string { return ''; }
}

class DateTimeZone
{
    public const AFRICA = 1;
    public const AMERICA = 2;
    public const ANTARCTICA = 4;
    public const ARCTIC = 8;
    public const ASIA = 16;
    public const ATLANTIC = 32;
    public const AUSTRALIA = 64;
    public const EUROPE = 128;
    public const INDIAN = 256;
    public const PACIFIC = 512;
    public const UTC = 1024;
    public const ALL = 2047;
    public const ALL_WITH_BC = 4095;
    public const PER_COUNTRY = 4096;

    public function __construct(string $timezone) {}

    public function getLocation(): array|false { return false; }

    public function getName(): string { return ''; }

    public function getOffset(DateTimeInterface $datetime): int { return 0; }

    public function getTransitions(int $timestampBegin = PHP_INT_MIN, int $timestampEnd = PHP_INT_MAX): array|false { return false; }

    public static function listAbbreviations(): array { return []; }

    public static function listIdentifiers(int $timezoneGroup = DateTimeZone::ALL, ?string $countryCode = null): array { return []; }
}

class DateInterval
{
    public int $y;
    public int $m;
    public int $d;
    public int $h;
    public int $i;
    public int $s;
    public float $f;
    public int $invert;
    public mixed $days;

    public function __construct(string $duration) {}

    public static function createFromDateString(string $datetime): static|false { return new static('P0D'); }

    public function format(string $format): string { return ''; }
}

/**
 * @template TDate of DateTimeInterface
 * @implements IteratorAggregate<int, TDate>
 */
class DatePeriod implements IteratorAggregate
{
    public const EXCLUDE_START_DATE = 1;
    public const INCLUDE_END_DATE = 2;

    /**
     * @param TDate $start
     */
    public function __construct(
        DateTimeInterface $start,
        DateInterval $interval,
        int|DateTimeInterface $end,
        int $options = 0,
    ) {}

    public function getDateInterval(): DateInterval { return new DateInterval('P0D'); }

    /** @return DateTimeInterface|null */
    public function getEndDate(): ?DateTimeInterface { return null; }

    /** @return DateTimeInterface */
    public function getStartDate(): DateTimeInterface { return new DateTime(); }

    public function getRecurrences(): ?int { return null; }

    /** @return \Iterator<int, TDate> */
    public function getIterator(): \Traversable { return new \ArrayIterator([]); }
}
