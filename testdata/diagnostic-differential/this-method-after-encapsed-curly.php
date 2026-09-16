<?php

namespace App\Tests\Unit\Module\Shift\DTO;

final class EmployeeCalendarDataTest
{
    public function testBuild(): void
    {
        $label = 'day';
        $msg = "label {$label}";
        $this->createDayAvailabilityData();
    }

    /**
     * @param ExistingAssignmentData[] $existingAssignments
     */
    private function createDayAvailabilityData(): void
    {
    }
}
