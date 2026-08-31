<?php
class FunctionService {}

function makeFunctionService(): FunctionService
{
    return new FunctionService();
}

makeFunctionService()->missing();
