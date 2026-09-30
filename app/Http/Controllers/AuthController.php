<?php

namespace AppHttpControllers;

use AppModelsUser;
use AppServicesOidcService;
use AppServicesProvisioningService;
use IlluminateHttpRedirectResponse;
use IlluminateSupportStr;
use Throwable;

class AuthController extends Controller
{
    private const USERNAME_PATTERN = '/^(?:[a-z]|[a-z][a-z0-9-]{0,30}[a-z0-9])$/';

    public function loginFromOidc(): RedirectResponse
    {
        try {
            $claims = app(OidcService::class)->authenticate();
            $username = Str::lower($claims['preferred_username'] ?? $claims['nickname'] ?? '');

            if (!preg_match(self::USERNAME_PATTERN, $username)) {
                abort(403, 'Your Authentik username is not valid for a People account.');
            }

            $groups = $claims['groups'] ?? [];
            if (is_string($groups)) {
                $groups = [$groups];
            }

            if (!in_array(config('people.authentik_group'), $groups, true)) {
                abort(403, 'You are not a member of the People group.');
            }

            $existingByUsername = User::query()
                ->where('username', $username)
                ->where('authentik_subject', '!=', $claims['sub'])
                ->exists();

            if ($existingByUsername) {
                abort(409, 'This People username is already assigned to another identity.');
            }

            $user = User::firstOrCreate(
                ['authentik_subject' => $claims['sub']],
                [
                    'username' => $username,
                    'email' => $claims['email'] ?? null,
                    'display_name' => $claims['name'] ?? $username,
                    'status' => 'provisioning',
                    'is_admin' => false,
                ],
            );

            if (in_array($user->status, ['suspended', 'deleted'], true)) {
                abort(403, 'This People account is disabled.');
            }

            $user->update([
                'email' => $claims['email'] ?? $user->email,
                'display_name' => $claims['name'] ?? $user->display_name,
                'is_admin' => in_array(config('people.authentik_admin_group'), $groups, true),
            ]);

            if ($user->status === 'provisioning') {
                app(ProvisioningService::class)->createUser($user);
                $user->update(['status' => 'active']);
            }

            session()->regenerate();
            session(['people_user_id' => $user->id]);

            return redirect()->route('dashboard');
        } catch (Throwable $e) {
            report($e);

            return redirect()->route('home')->with(
                'error',
                $e instanceof SymfonyComponentHttpKernelExceptionHttpExceptionInterface
                    ? $e->getMessage()
                    : 'Login failed. Check the dashboard logs.',
            );
        }
    }

    public function logout(IlluminateHttpRequest $request): RedirectResponse
    {
        $request->session()->invalidate();
        $request->session()->regenerateToken();

        return redirect()->route('home');
    }
}
