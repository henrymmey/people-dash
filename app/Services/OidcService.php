<?php

namespace App\Services;

use Jumbojett\OpenIDConnectClient;
use RuntimeException;

class OidcService
{
    public function authenticate(): array
    {
        $oidc=new OpenIDConnectClient(config('people.oidc.issuer'),config('people.oidc.client_id'),config('people.oidc.client_secret'));
        $oidc->setRedirectURL(config('people.oidc.redirect_uri')); $oidc->addScope('openid'); $oidc->addScope('profile'); $oidc->addScope('email'); $oidc->setCodeChallengeMethod('S256');
        if(!$oidc->authenticate())throw new RuntimeException('OIDC authentication failed.');
        $claims=(array)$oidc->requestUserInfo(); if(empty($claims['sub']))throw new RuntimeException('Authentik did not return a subject.'); return $claims;
    }
}
