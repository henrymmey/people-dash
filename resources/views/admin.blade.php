@extends('layouts.app')

@section('content')
    <div class="card">
        <h1>Administration</h1>
        <p class="muted">Manage People accounts and inspect recent provisioning operations.</p>
    </div>

    <div class="card">
        <h2>Users</h2>
        <div style="overflow-x:auto">
            <table style="width:100%;border-collapse:collapse">
                <thead>
                    <tr>
                        <th align="left">Username</th>
                        <th align="left">Email</th>
                        <th align="left">Status</th>
                        <th align="left">Keys</th>
                        <th align="left">Actions</th>
                    </tr>
                </thead>
                <tbody>
                @foreach($users as $managed)
                    <tr>
                        <td style="padding:10px 8px 10px 0">
                            <strong>{{ $managed->username }}</strong>
                            @if($managed->id === session('people_user_id'))
                                <span class="muted">(you)</span>
                            @endif
                        </td>
                        <td style="padding:10px 8px">{{ $managed->email }}</td>
                        <td style="padding:10px 8px">
                            <span class="button" style="padding:6px 10px;cursor:default;{{ $managed->status === 'active' ? '' : ($managed->status === 'suspended' ? 'background:#9a6700' : 'background:#667085') }}">
                                {{ $managed->status }}
                            </span>
                        </td>
                        <td style="padding:10px 8px">{{ $managed->ssh_keys_count }}</td>
                        <td style="padding:10px 0 10px 8px">
                            @if($managed->id === session('people_user_id'))
                                <span class="muted">Protected account</span>
                            @elseif($managed->status === 'active')
                                <form method="post" action="{{ route('admin.suspend', $managed) }}" style="display:inline">
                                    @csrf
                                    <button class="button" type="submit">Suspend</button>
                                </form>
                                <form method="post" action="{{ route('admin.delete', $managed) }}" style="display:inline;margin-left:6px" onsubmit="return confirm('Permanently delete this People account and its home directory?');">
                                    @csrf
                                    @method('DELETE')
                                    <button class="button danger" type="submit">Delete</button>
                                </form>
                            @elseif($managed->status === 'suspended')
                                <form method="post" action="{{ route('admin.resume', $managed) }}" style="display:inline">
                                    @csrf
                                    <button class="button" type="submit">Resume</button>
                                </form>
                                <form method="post" action="{{ route('admin.delete', $managed) }}" style="display:inline;margin-left:6px" onsubmit="return confirm('Permanently delete this People account and its home directory?');">
                                    @csrf
                                    @method('DELETE')
                                    <button class="button danger" type="submit">Delete</button>
                                </form>
                            @else
                                <span class="muted">No actions</span>
                            @endif
                        </td>
                    </tr>
                @endforeach
                </tbody>
            </table>
        </div>
    </div>

    <div class="card">
        <h2>Recent provisioning jobs</h2>
        @forelse($jobs as $job)
            <div style="padding:10px 0;border-bottom:1px solid #eee">
                <strong>{{ $job->type }}</strong>
                · {{ $job->user->username }}
                · {{ $job->status }}
                <span class="muted">· {{ $job->created_at }}</span>
                @if($job->error)
                    <div class="error" style="margin-top:8px">{{ $job->error }}</div>
                @endif
            </div>
        @empty
            <p class="muted">No provisioning jobs yet.</p>
        @endforelse
    </div>
@endsection
