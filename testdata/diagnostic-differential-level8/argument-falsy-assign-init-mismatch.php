<?php

class ResetPassword
{
}

function save(ResetPassword $reset): void
{
}

function findReset(?ResetPassword $candidate): ?ResetPassword
{
	return $candidate;
}

function run(?ResetPassword $candidate): void
{
	$passwordReset = findReset($candidate);
    save($passwordReset);
}
