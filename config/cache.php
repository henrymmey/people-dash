<?php
return ['default'=>env('CACHE_STORE','array'),'stores'=>['array'=>['driver'=>'array','serialize'=>false],'database'=>['driver'=>'database','connection'=>env('DB_CACHE_CONNECTION'),'table'=>env('DB_CACHE_TABLE','cache'),'lock_connection'=>env('DB_CACHE_LOCK_CONNECTION'),'lock_table'=>env('DB_CACHE_LOCK_TABLE','cache_locks')]],'prefix'=>env('CACHE_PREFIX','people_cache')];
