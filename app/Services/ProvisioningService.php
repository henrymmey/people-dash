<?php

namespace App\Services;

use App\Models\ProvisioningJob;
use App\Models\User;
use Illuminate\Support\Facades\Http;
use RuntimeException;

class ProvisioningService
{
    private function client()
    {
        return Http::baseUrl(rtrim(config('people.agent_url'), '/'))
            ->withToken(config('people.agent_token'))
            ->acceptJson()
            ->asJson()
            ->connectTimeout(5)
            ->timeout(15);
    }

    public function createUser(User $user): ProvisioningJob
    {
        return $this->run($user, 'CREATE_USER', [
            'username' => $user->username,
            'email' => $user->email,
        ]);
    }

    public function deleteUser(User $user): ProvisioningJob
    {
        return $this->run($user, 'DELETE_USER', [
            'username' => $user->username,
        ]);
    }

    public function suspendUser(User $user): ProvisioningJob
    {
        return $this->run($user, 'SUSPEND_USER', [
            'username' => $user->username,
        ]);
    }

    public function resumeUser(User $user): ProvisioningJob
    {
        return $this->run($user, 'RESUME_USER', [
            'username' => $user->username,
        ]);
    }

    public function addKey(User $user, string $publicKey): ProvisioningJob
    {
        return $this->run($user, 'ADD_SSH_KEY', [
            'username' => $user->username,
            'public_key' => $publicKey,
        ]);
    }

    public function removeKey(User $user, string $publicKey): ProvisioningJob
    {
        return $this->run($user, 'REMOVE_SSH_KEY', [
            'username' => $user->username,
            'public_key' => $publicKey,
        ]);
    }

    public function storage(User $user): array
    {
        $response = $this->client()
            ->get('/v1/users/' . rawurlencode($user->username) . '/storage');

        if (!$response->successful()) {
            throw new RuntimeException('Provisioning agent storage request failed.');
        }

        return $response->json();
    }

    private function run(User $user, string $type, array $payload): ProvisioningJob
    {
        $job = ProvisioningJob::create([
            'user_id' => $user->id,
            'type' => $type,
            'status' => 'running',
            'payload' => $payload,
        ]);

        try {
            [$method, $path, $body] = match ($type) {
                'CREATE_USER' => ['post', '/v1/users', $payload],
                'DELETE_USER' => ['delete', '/v1/users/' . rawurlencode($user->username), null],
                'SUSPEND_USER' => ['post', '/v1/users/' . rawurlencode($user->username) . '/suspend', null],
                'RESUME_USER' => ['delete', '/v1/users/' . rawurlencode($user->username) . '/suspend', null],
                'ADD_SSH_KEY' => [
                    'post',
                    '/v1/users/' . rawurlencode($user->username) . '/ssh-keys',
                    ['public_key' => $payload['public_key']],
                ],
                'REMOVE_SSH_KEY' => [
                    'delete',
                    '/v1/users/' . rawurlencode($user->username) . '/ssh-keys',
                    ['public_key' => $payload['public_key']],
                ],
                default => throw new RuntimeException('Unknown provisioning operation.'),
            };

            $request = $this->client();

            $response = match ($method) {
                'post' => $request->post($path, $body),
                'delete' => $body === null
                    ? $request->delete($path)
                    : $request
                        ->withBody(json_encode($body, JSON_THROW_ON_ERROR), 'application/json')
                        ->delete($path),
                default => throw new RuntimeException('Unsupported provisioning HTTP method.'),
            };

            if (!$response->successful()) {
                throw new RuntimeException(
                    'Agent returned HTTP ' . $response->status() . ': ' . $response->body(),
                );
            }

            $job->update([
                'status' => 'completed',
                'completed_at' => now(),
            ]);

            return $job;
        } catch (\Throwable $e) {
            $job->update([
                'status' => 'failed',
                'error' => mb_substr($e->getMessage(), 0, 2000),
                'completed_at' => now(),
            ]);

            throw $e;
        }
    }
}
