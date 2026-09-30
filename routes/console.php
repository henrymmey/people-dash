<?php
use Illuminate\Support\Facades\Artisan;
Artisan::command('people:health', function(){ $this->info('People Dash is healthy.'); });
