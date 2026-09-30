<?php
namespace App\Http\Middleware;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;
use App\Models\User;
class AdminOnly { public function handle(Request $request,Closure $next): Response { $user=User::find(session('people_user_id')); abort_unless($user?->is_admin,403); return $next($request); } }
