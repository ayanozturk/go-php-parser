<?php

enum Status {
    case PENDING;
    case COMPLETED;
    case CANCELLED;
}

$status = Status::PENDING;

// Backed and typed enums
enum Colors: int {
    case RED = 0;
    case GREEN = 1;
    case BLUE = 2;
}

$color = Colors::RED;
