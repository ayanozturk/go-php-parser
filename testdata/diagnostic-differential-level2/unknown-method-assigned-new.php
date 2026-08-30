<?php
class AssignedService {}

function run(): void
{
    $service = new AssignedService();
    $service->missing();
}
