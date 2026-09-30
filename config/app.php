<?php

use IlluminateSupportFacadesFacade;
use IlluminateSupportServiceProvider;

return [
    'name' => env('APP_NAME', 'People Dash'),
    'env' => env('APP_ENV', 'production'),
    'debug' => (bool) env('APP_DEBUG', false),
    'url' => env('APP_URL', 'https://p.meyerbrief.de'),
    'timezone' => 'Europe/Berlin',
    'locale' => env('APP_LOCALE', 'en'),
    'fallback_locale' => env('APP_FALLBACK_LOCALE', 'en'),
    'faker_locale' => env('APP_FAKER_LOCALE', 'de_DE'),

    'key' => env('APP_KEY'),
    'cipher' => 'AES-256-CBC',
    'previous_keys' => array_filter(
        explode(',', (string) env('APP_PREVIOUS_KEYS', '')),
    ),

    'maintenance' => [
        'driver' => env('APP_MAINTENANCE_DRIVER', 'file'),
        'store' => env('APP_MAINTENANCE_STORE', 'database'),
    ],

    'providers' => ServiceProvider::defaultProviders()
        ->merge([
            // Application service providers.
        ])
        ->toArray(),

    'aliases' => Facade::defaultAliases()
        ->merge([
            // Application aliases.
        ])
        ->toArray(),
];
