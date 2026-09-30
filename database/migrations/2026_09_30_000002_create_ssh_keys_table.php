<?php
use Illuminate\Database\Migrations\Migration; use Illuminate\Database\Schema\Blueprint; use Illuminate\Support\Facades\Schema;
return new class extends Migration { public function up(): void {Schema::create('ssh_keys',function(Blueprint $table){$table->id();$table->foreignId('user_id')->constrained()->cascadeOnDelete();$table->string('name',80);$table->text('public_key');$table->string('fingerprint',100);$table->timestamps();$table->unique(['user_id','fingerprint']);});} public function down():void{Schema::dropIfExists('ssh_keys');} };
