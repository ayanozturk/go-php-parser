<?php

class PDOException extends RuntimeException
{
}

class PDO
{
    const PARAM_NULL = 0;
    const PARAM_BOOL = 5;
    const PARAM_INT = 1;
    const PARAM_STR = 2;
    const PARAM_LOB = 3;
    const PARAM_STMT = 4;
    const PARAM_INPUT_OUTPUT = 2147483648;
    const PARAM_STR_NATL = 1073741824;
    const PARAM_STR_CHAR = 536870912;
    const PARAM_EVT_ALLOC = 0;
    const PARAM_EVT_FREE = 1;
    const PARAM_EVT_EXEC_PRE = 2;
    const PARAM_EVT_EXEC_POST = 3;
    const PARAM_EVT_FETCH_PRE = 4;
    const PARAM_EVT_FETCH_POST = 5;
    const PARAM_EVT_NORMALIZE = 6;
    const FETCH_DEFAULT = 0;
    const FETCH_LAZY = 1;
    const FETCH_ASSOC = 2;
    const FETCH_NUM = 3;
    const FETCH_BOTH = 4;
    const FETCH_OBJ = 5;
    const FETCH_BOUND = 6;
    const FETCH_COLUMN = 7;
    const FETCH_CLASS = 8;
    const FETCH_INTO = 9;
    const FETCH_FUNC = 10;
    const FETCH_GROUP = 32;
    const FETCH_UNIQUE = 64;
    const FETCH_KEY_PAIR = 12;
    const FETCH_CLASSTYPE = 128;
    const FETCH_SERIALIZE = 512;
    const FETCH_PROPS_LATE = 256;
    const FETCH_NAMED = 11;
    const ATTR_AUTOCOMMIT = 0;
    const ATTR_PREFETCH = 1;
    const ATTR_TIMEOUT = 2;
    const ATTR_ERRMODE = 3;
    const ATTR_SERVER_VERSION = 4;
    const ATTR_CLIENT_VERSION = 5;
    const ATTR_SERVER_INFO = 6;
    const ATTR_CONNECTION_STATUS = 7;
    const ATTR_CASE = 8;
    const ATTR_CURSOR_NAME = 9;
    const ATTR_CURSOR = 10;
    const ATTR_ORACLE_NULLS = 11;
    const ATTR_PERSISTENT = 12;
    const ATTR_STATEMENT_CLASS = 13;
    const ATTR_FETCH_TABLE_NAMES = 14;
    const ATTR_FETCH_CATALOG_NAMES = 15;
    const ATTR_DRIVER_NAME = 16;
    const ATTR_STRINGIFY_FETCHES = 17;
    const ATTR_MAX_COLUMN_LEN = 18;
    const ATTR_EMULATE_PREPARES = 20;
    const ATTR_DEFAULT_FETCH_MODE = 19;
    const ATTR_DEFAULT_STR_PARAM = 21;
    const ERRMODE_SILENT = 0;
    const ERRMODE_WARNING = 1;
    const ERRMODE_EXCEPTION = 2;
    const CASE_NATURAL = 0;
    const CASE_LOWER = 2;
    const CASE_UPPER = 1;
    const NULL_NATURAL = 0;
    const NULL_EMPTY_STRING = 1;
    const NULL_TO_STRING = 2;
    const ERR_NONE = '00000';
    const FETCH_ORI_NEXT = 0;
    const FETCH_ORI_PRIOR = 1;
    const FETCH_ORI_FIRST = 2;
    const FETCH_ORI_LAST = 3;
    const FETCH_ORI_ABS = 4;
    const FETCH_ORI_REL = 5;
    const CURSOR_FWDONLY = 0;
    const CURSOR_SCROLL = 1;
    const DBLIB_ATTR_CONNECTION_TIMEOUT = 1000;
    const DBLIB_ATTR_QUERY_TIMEOUT = 1001;
    const DBLIB_ATTR_STRINGIFY_UNIQUEIDENTIFIER = 1002;
    const DBLIB_ATTR_VERSION = 1003;
    const DBLIB_ATTR_TDS_VERSION = 1004;
    const DBLIB_ATTR_SKIP_EMPTY_ROWSETS = 1005;
    const DBLIB_ATTR_DATETIME_CONVERT = 1006;
    const MYSQL_ATTR_USE_BUFFERED_QUERY = 1000;
    const MYSQL_ATTR_LOCAL_INFILE = 1001;
    const MYSQL_ATTR_INIT_COMMAND = 1002;
    const MYSQL_ATTR_COMPRESS = 1003;
    const MYSQL_ATTR_DIRECT_QUERY = 20;
    const MYSQL_ATTR_FOUND_ROWS = 1004;
    const MYSQL_ATTR_IGNORE_SPACE = 1005;
    const MYSQL_ATTR_SSL_KEY = 1006;
    const MYSQL_ATTR_SSL_CERT = 1007;
    const MYSQL_ATTR_SSL_CA = 1008;
    const MYSQL_ATTR_SSL_CAPATH = 1009;
    const MYSQL_ATTR_SSL_CIPHER = 1010;
    const MYSQL_ATTR_SERVER_PUBLIC_KEY = 1011;
    const MYSQL_ATTR_MULTI_STATEMENTS = 1012;
    const MYSQL_ATTR_SSL_VERIFY_SERVER_CERT = 1013;
    const MYSQL_ATTR_LOCAL_INFILE_DIRECTORY = 1014;
    const ODBC_ATTR_USE_CURSOR_LIBRARY = 1000;
    const ODBC_ATTR_ASSUME_UTF8 = 1001;
    const ODBC_SQL_USE_IF_NEEDED = 0;
    const ODBC_SQL_USE_DRIVER = 2;
    const ODBC_SQL_USE_ODBC = 1;
    const PGSQL_ATTR_DISABLE_PREPARES = 1000;
    const PGSQL_TRANSACTION_IDLE = 0;
    const PGSQL_TRANSACTION_ACTIVE = 1;
    const PGSQL_TRANSACTION_INTRANS = 2;
    const PGSQL_TRANSACTION_INERROR = 3;
    const PGSQL_TRANSACTION_UNKNOWN = 4;
    const SQLITE_DETERMINISTIC = 2048;
    const SQLITE_ATTR_OPEN_FLAGS = 1000;
    const SQLITE_OPEN_READONLY = 1;
    const SQLITE_OPEN_READWRITE = 2;
    const SQLITE_OPEN_CREATE = 4;
    const SQLITE_ATTR_READONLY_STATEMENT = 1001;
    const SQLITE_ATTR_EXTENDED_RESULT_CODES = 1002;
    public function __construct(string $dsn, ?string $username = NULL, ?string $password = NULL, ?array $options = NULL) { return null; }

    static public function connect(string $dsn, ?string $username = NULL, ?string $password = NULL, ?array $options = NULL): static { return new static(); }

    public function beginTransaction() { return null; }

    public function commit() { return null; }

    public function errorCode() { return null; }

    public function errorInfo() { return null; }

    public function exec(string $statement) { return null; }

    public function getAttribute(int $attribute) { return null; }

    static public function getAvailableDrivers() { return null; }

    public function inTransaction() { return null; }

    public function lastInsertId(?string $name = NULL) { return null; }

    public function prepare(string $query, array $options = []) { return null; }

    public function query(string $query, ?int $fetchMode = NULL, mixed ...$fetchModeArgs) { return null; }

    public function quote(string $string, int $type = 2) { return null; }

    public function rollBack() { return null; }

    public function setAttribute(int $attribute, mixed $value) { return null; }
}

class PDOStatement implements IteratorAggregate, Traversable
{
    public function bindColumn(string|int $column, mixed &$var, int $type = 2, int $maxLength = 0, mixed $driverOptions = NULL) { return null; }

    public function bindParam(string|int $param, mixed &$var, int $type = 2, int $maxLength = 0, mixed $driverOptions = NULL) { return null; }

    public function bindValue(string|int $param, mixed $value, int $type = 2) { return null; }

    public function closeCursor() { return null; }

    public function columnCount() { return null; }

    public function debugDumpParams() { return null; }

    public function errorCode() { return null; }

    public function errorInfo() { return null; }

    public function execute(?array $params = NULL) { return null; }

    public function fetch(int $mode = 0, int $cursorOrientation = 0, int $cursorOffset = 0) { return null; }

    public function fetchAll(int $mode = 0, mixed ...$args) { return null; }

    public function fetchColumn(int $column = 0) { return null; }

    public function fetchObject(?string $class = 'stdClass', array $constructorArgs = []) { return null; }

    public function getAttribute(int $name) { return null; }

    public function getColumnMeta(int $column) { return null; }

    public function nextRowset() { return null; }

    public function rowCount() { return null; }

    public function setAttribute(int $attribute, mixed $value) { return null; }

    public function setFetchMode(int $mode, mixed ...$args) { return null; }

    public function getIterator(): Iterator { return null; }
}

class PDORow
{
}

