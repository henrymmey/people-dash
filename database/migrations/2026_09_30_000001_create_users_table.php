<?php
use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;
return new class extends Migration { public function up(): void { Schema::create('users',function(Blueprint $table){$table->id();$table->string('authentik_subject')->unique();$table->string('username',32)->unique();$table->string('email')->nullable();$table->string('display_name')->nullable();$table->string('status')->default('provisioning');$table->boolean('is_admin')->default(false);$table->timestamps();}); } public function down(): void {Schema::dropIfExists('users');} };
