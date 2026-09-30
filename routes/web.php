<?php
use App\Http\Controllers\AuthController; use App\Http\Controllers\DashboardController; use Illuminate\Support\Facades\Route;
Route::get('/',fn()=>view('home'))->name('home');
Route::get('/auth/login',[AuthController::class,'loginFromOidc'])->name('auth.login'); Route::get('/auth/callback',[AuthController::class,'loginFromOidc'])->name('auth.callback'); Route::post('/auth/logout',[AuthController::class,'logout'])->name('auth.logout');
Route::middleware(App\Http\Middleware\AuthenticatePeople::class)->group(function(){
 Route::get('/dashboard',[DashboardController::class,'index'])->name('dashboard'); Route::get('/ssh-keys',[DashboardController::class,'keys'])->name('keys'); Route::post('/ssh-keys',[DashboardController::class,'addKey'])->name('keys.add'); Route::delete('/ssh-keys/{key}',[DashboardController::class,'removeKey'])->name('keys.remove');
 Route::middleware(App\Http\Middleware\AdminOnly::class)->group(function(){Route::get('/admin',[DashboardController::class,'admin'])->name('admin');Route::post('/admin/users/{managed}/suspend',[DashboardController::class,'suspend'])->name('admin.suspend');Route::post('/admin/users/{managed}/resume',[DashboardController::class,'resume'])->name('admin.resume');Route::delete('/admin/users/{managed}',[DashboardController::class,'delete'])->name('admin.delete');});
});
