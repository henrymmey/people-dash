<?php
return [
 'name'=>env('APP_NAME','People Dash'),'env'=>env('APP_ENV','production'),'debug'=>(bool)env('APP_DEBUG',false),'url'=>env('APP_URL','https://p.meyerbrief.de'),'timezone'=>'Europe/Berlin','locale'=>'en','fallback_locale'=>'en','faker_locale'=>'de_DE','key'=>env('APP_KEY'),'cipher'=>'AES-256-CBC',
 'maintenance'=>['driver'=>'file'],
 'providers'=>[], 'aliases'=>[],
];
