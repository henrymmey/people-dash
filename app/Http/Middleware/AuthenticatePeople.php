<?php
namespace App\Http\Middleware;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;
class AuthenticatePeople {
 public function handle(Request $request, Closure $next): Response {
   if($request->routeIs('auth.*') || $request->is('up')) return $next($request);
   if(!session()->has('people_user_id')) return redirect()->route('auth.login');
   return $next($request);
 }
}
