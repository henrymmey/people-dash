<?php

namespace App\Http\Middleware;

use App\Models\User;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class AuthenticatePeople
{
    public function handle(Request $request, Closure $next): Response
    {
        if ($request->routeIs('auth.*') || $request->is('up')) {
            return $next($request);
        }

        $id = $request->session()->get('people_user_id');

        if (!$id) {
            return redirect()->route('auth.login');
        }

        $user = User::find($id);

        if (!$user || $user->status !== 'active') {
            $request->session()->forget('people_user_id');
            $request->session()->invalidate();
            $request->session()->regenerateToken();

            return redirect()->route('home')->with('error', 'Your People account is currently disabled.');
        }

        return $next($request);
    }
}
