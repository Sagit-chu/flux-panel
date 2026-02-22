<?php
/**
 * FLVX WHMCS Provisioning Module
 * 
 * This module provides integration between WHMCS and FLVX panel for
 * traffic forwarding service provisioning.
 * 
 * @package   FLVX WHMCS Module
 * @version   1.0.0
 */

if (!defined('WHMCS')) {
    die('This file cannot be accessed directly');
}

require_once __DIR__ . '/lib/FlvxApiClient.php';

use FLVX\FlvxApiClient;

// ==========================================
// Module Meta Data
// ==========================================

function flvx_MetaData()
{
    return [
        'DisplayName' => 'FLVX Traffic Forwarding',
        'APIVersion' => '1.1',
        'RequiresServer' => true,
        'DefaultNonSSLPort' => '6365',
        'DefaultSSLPort' => '6365',
        'ServiceSingleSignOnLabel' => 'Login to FLVX Panel',
    ];
}

// ==========================================
// Product Configuration Options
// ==========================================

function flvx_ConfigOptions()
{
    return [
        'Traffic Quota (GB)' => [
            'Type' => 'text',
            'Size' => '10',
            'Default' => '100',
            'Description' => 'Traffic quota in gigabytes',
        ],
        'Max Forwards' => [
            'Type' => 'text',
            'Size' => '10',
            'Default' => '10',
            'Description' => 'Maximum number of port forwards allowed',
        ],
        'Tunnel ID' => [
            'Type' => 'text',
            'Size' => '10',
            'Default' => '',
            'Description' => 'Default tunnel ID to assign (leave empty to use server setting)',
        ],
        'Speed Limit ID' => [
            'Type' => 'text',
            'Size' => '10',
            'Default' => '',
            'Description' => 'Speed limit ID (leave empty for unlimited)',
        ],
        'Expiry Days' => [
            'Type' => 'text',
            'Size' => '10',
            'Default' => '30',
            'Description' => 'Service validity in days',
        ],
    ];
}

// ==========================================
// Helper Functions
// ==========================================

function flvx_getApiClient($params)
{
    $apiUrl = $params['serverhostname'] ?? '';
    $username = $params['serverusername'] ?? '';
    $password = $params['serverpassword'] ?? '';

    if (empty($apiUrl) || empty($username) || empty($password)) {
        throw new Exception('FLVX API credentials not configured');
    }

    if (isset($params['serversecure']) && $params['serversecure']) {
        $apiUrl = 'https://' . $apiUrl;
    } else {
        $apiUrl = 'http://' . $apiUrl;
    }

    return new FlvxApiClient($apiUrl, $username, $password);
}

function flvx_getServiceCustomField($serviceId, $fieldName)
{
    $result = \WHMCS\Database\Capsule::table('tblcustomfields')
        ->join('tblcustomfieldsvalues', 'tblcustomfields.id', '=', 'tblcustomfieldsvalues.fieldid')
        ->where('tblcustomfields.type', 'product')
        ->where('tblcustomfields.fieldname', 'like', $fieldName . '%')
        ->where('tblcustomfieldsvalues.relid', $serviceId)
        ->first(['tblcustomfieldsvalues.value']);

    return $result ? $result->value : null;
}

function flvx_setServiceCustomField($serviceId, $fieldName, $value)
{
    $field = \WHMCS\Database\Capsule::table('tblcustomfields')
        ->where('type', 'product')
        ->where('fieldname', 'like', $fieldName . '%')
        ->first(['id']);

    if (!$field) {
        return false;
    }

    $exists = \WHMCS\Database\Capsule::table('tblcustomfieldsvalues')
        ->where('fieldid', $field->id)
        ->where('relid', $serviceId)
        ->first();

    if ($exists) {
        \WHMCS\Database\Capsule::table('tblcustomfieldsvalues')
            ->where('fieldid', $field->id)
            ->where('relid', $serviceId)
            ->update(['value' => $value]);
    } else {
        \WHMCS\Database\Capsule::table('tblcustomfieldsvalues')
            ->insert([
                'fieldid' => $field->id,
                'relid' => $serviceId,
                'value' => $value,
            ]);
    }

    return true;
}

function flvx_generateUsername($params)
{
    $email = $params['clientsdetails']['email'] ?? '';
    $domain = $params['domain'] ?? '';
    $serviceId = $params['serviceid'] ?? time();
    
    if (!empty($domain)) {
        return 'u' . preg_replace('/[^a-z0-9]/', '', strtolower(explode('.', $domain)[0]));
    }
    
    if (!empty($email)) {
        $localPart = explode('@', $email)[0];
        return 'u' . preg_replace('/[^a-z0-9]/', '', strtolower($localPart)) . substr($serviceId, -4);
    }
    
    return 'u' . $serviceId;
}

function flvx_log($action, $requestData, $responseData, $status = 'success')
{
    logModuleCall(
        'flvx',
        $action,
        is_array($requestData) ? json_encode($requestData) : $requestData,
        is_array($responseData) ? json_encode($responseData) : $responseData,
        '',
        $status
    );
}

// ==========================================
// Core Provisioning Functions
// ==========================================

function flvx_CreateAccount($params)
{
    try {
        $api = flvx_getApiClient($params);
        $serviceId = $params['serviceid'];
        
        $username = flvx_generateUsername($params);
        $password = FlvxApiClient::generatePassword(12);
        
        $configTrafficGb = (int)($params['configoption1'] ?? 100);
        $configMaxForwards = (int)($params['configoption2'] ?? 10);
        $configTunnelId = $params['configoption3'] ?? '';
        $configSpeedId = $params['configoption4'] ?? '';
        $configExpiryDays = (int)($params['configoption5'] ?? 30);
        
        $tunnelId = !empty($configTunnelId) ? $configTunnelId : ($params['serveraccesshash'] ?? '');
        if (empty($tunnelId)) {
            return ['error' => 'No tunnel ID configured. Please set it in product config or server settings.'];
        }
        
        $existingUser = $api->getUserByUsername($username);
        $flvxUserId = null;
        
        if ($existingUser !== false) {
            $flvxUserId = $existingUser['id'];
            flvx_log('CreateAccount', ['username' => $username], 'User already exists: ' . $flvxUserId);
        } else {
            $expiryTime = FlvxApiClient::nowMs() + FlvxApiClient::daysToMs($configExpiryDays);
            
            $userResult = $api->createUser($username, $password, [
                'flow' => $configTrafficGb,
                'num' => $configMaxForwards,
                'expTime' => $expiryTime,
                'status' => 1,
            ]);
            
            flvx_log('CreateAccount - CreateUser', [
                'username' => $username,
                'flow' => $configTrafficGb,
                'expTime' => $expiryTime,
            ], $userResult);
            
            if ($userResult === false) {
                return ['error' => 'Failed to create FLVX user: ' . $api->getLastError()];
            }
            
            $flvxUser = $api->getUserByUsername($username);
            if ($flvxUser === false) {
                return ['error' => 'Failed to retrieve created user'];
            }
            $flvxUserId = $flvxUser['id'];
        }
        
        $assignOptions = [
            'flow' => FlvxApiClient::gbToBytes($configTrafficGb),
            'num' => $configMaxForwards,
            'expTime' => FlvxApiClient::nowMs() + FlvxApiClient::daysToMs($configExpiryDays),
            'status' => 1,
        ];
        
        if (!empty($configSpeedId)) {
            $assignOptions['speedId'] = (int)$configSpeedId;
        }
        
        $assignResult = $api->assignTunnelToUser($flvxUserId, (int)$tunnelId, $assignOptions);
        
        flvx_log('CreateAccount - AssignTunnel', [
            'userId' => $flvxUserId,
            'tunnelId' => $tunnelId,
            'options' => $assignOptions,
        ], $assignResult);
        
        if ($assignResult === false) {
            return ['error' => 'Failed to assign tunnel: ' . $api->getLastError()];
        }
        
        $userTunnels = $api->listUserTunnels($flvxUserId);
        $userTunnelId = null;
        if ($userTunnels !== false) {
            foreach ($userTunnels as $ut) {
                if (isset($ut['tunnelId']) && $ut['tunnelId'] == $tunnelId) {
                    $userTunnelId = $ut['id'];
                    break;
                }
            }
        }
        
        flvx_setServiceCustomField($serviceId, 'flvx_user_id', $flvxUserId);
        flvx_setServiceCustomField($serviceId, 'flvx_username', $username);
        flvx_setServiceCustomField($serviceId, 'flvx_password', $password);
        flvx_setServiceCustomField($serviceId, 'flvx_tunnel_id', $tunnelId);
        if ($userTunnelId) {
            flvx_setServiceCustomField($serviceId, 'flvx_user_tunnel_id', $userTunnelId);
        }
        
        return 'success';
        
    } catch (Exception $e) {
        flvx_log('CreateAccount', $params, $e->getMessage(), 'error');
        return ['error' => $e->getMessage()];
    }
}

function flvx_SuspendAccount($params)
{
    try {
        $api = flvx_getApiClient($params);
        $serviceId = $params['serviceid'];
        
        $userTunnelId = flvx_getServiceCustomField($serviceId, 'flvx_user_tunnel_id');
        
        if (empty($userTunnelId)) {
            return ['error' => 'User tunnel ID not found. Service may not be provisioned correctly.'];
        }
        
        $result = $api->updateUserTunnel($userTunnelId, ['status' => 0]);
        
        flvx_log('SuspendAccount', ['userTunnelId' => $userTunnelId], $result);
        
        if ($result === false) {
            return ['error' => 'Failed to suspend service: ' . $api->getLastError()];
        }
        
        return 'success';
        
    } catch (Exception $e) {
        flvx_log('SuspendAccount', $params, $e->getMessage(), 'error');
        return ['error' => $e->getMessage()];
    }
}

function flvx_UnsuspendAccount($params)
{
    try {
        $api = flvx_getApiClient($params);
        $serviceId = $params['serviceid'];
        
        $userTunnelId = flvx_getServiceCustomField($serviceId, 'flvx_user_tunnel_id');
        
        if (empty($userTunnelId)) {
            return ['error' => 'User tunnel ID not found. Service may not be provisioned correctly.'];
        }
        
        $result = $api->updateUserTunnel($userTunnelId, ['status' => 1]);
        
        flvx_log('UnsuspendAccount', ['userTunnelId' => $userTunnelId], $result);
        
        if ($result === false) {
            return ['error' => 'Failed to unsuspend service: ' . $api->getLastError()];
        }
        
        return 'success';
        
    } catch (Exception $e) {
        flvx_log('UnsuspendAccount', $params, $e->getMessage(), 'error');
        return ['error' => $e->getMessage()];
    }
}

function flvx_TerminateAccount($params)
{
    try {
        $api = flvx_getApiClient($params);
        $serviceId = $params['serviceid'];
        
        $userTunnelId = flvx_getServiceCustomField($serviceId, 'flvx_user_tunnel_id');
        $flvxUserId = flvx_getServiceCustomField($serviceId, 'flvx_user_id');
        
        if (!empty($userTunnelId)) {
            $result = $api->removeUserTunnel($userTunnelId);
            flvx_log('TerminateAccount - RemoveTunnel', ['userTunnelId' => $userTunnelId], $result);
        }
        
        flvx_setServiceCustomField($serviceId, 'flvx_user_id', '');
        flvx_setServiceCustomField($serviceId, 'flvx_username', '');
        flvx_setServiceCustomField($serviceId, 'flvx_password', '');
        flvx_setServiceCustomField($serviceId, 'flvx_tunnel_id', '');
        flvx_setServiceCustomField($serviceId, 'flvx_user_tunnel_id', '');
        
        return 'success';
        
    } catch (Exception $e) {
        flvx_log('TerminateAccount', $params, $e->getMessage(), 'error');
        return ['error' => $e->getMessage()];
    }
}

// ==========================================
// Optional Provisioning Functions
// ==========================================

function flvx_ChangePassword($params)
{
    try {
        $api = flvx_getApiClient($params);
        $serviceId = $params['serviceid'];
        
        $flvxUserId = flvx_getServiceCustomField($serviceId, 'flvx_user_id');
        $username = flvx_getServiceCustomField($serviceId, 'flvx_username');
        
        if (empty($flvxUserId) || empty($username)) {
            return ['error' => 'User not provisioned'];
        }
        
        $newPassword = $params['password'];
        
        $result = $api->updateUser($flvxUserId, [
            'user' => $username,
            'pwd' => $newPassword,
        ]);
        
        flvx_log('ChangePassword', ['userId' => $flvxUserId], $result);
        
        if ($result === false) {
            return ['error' => 'Failed to change password: ' . $api->getLastError()];
        }
        
        flvx_setServiceCustomField($serviceId, 'flvx_password', $newPassword);
        
        return 'success';
        
    } catch (Exception $e) {
        flvx_log('ChangePassword', $params, $e->getMessage(), 'error');
        return ['error' => $e->getMessage()];
    }
}

function flvx_ChangePackage($params)
{
    try {
        $api = flvx_getApiClient($params);
        $serviceId = $params['serviceid'];
        
        $userTunnelId = flvx_getServiceCustomField($serviceId, 'flvx_user_tunnel_id');
        $flvxUserId = flvx_getServiceCustomField($serviceId, 'flvx_user_id');
        
        if (empty($userTunnelId) || empty($flvxUserId)) {
            return ['error' => 'Service not provisioned'];
        }
        
        $configTrafficGb = (int)($params['configoption1'] ?? 100);
        $configMaxForwards = (int)($params['configoption2'] ?? 10);
        $configSpeedId = $params['configoption4'] ?? '';
        $configExpiryDays = (int)($params['configoption5'] ?? 30);
        
        $tunnelOptions = [
            'flow' => FlvxApiClient::gbToBytes($configTrafficGb),
            'num' => $configMaxForwards,
            'expTime' => FlvxApiClient::nowMs() + FlvxApiClient::daysToMs($configExpiryDays),
        ];
        
        if (!empty($configSpeedId)) {
            $tunnelOptions['speedId'] = (int)$configSpeedId;
        }
        
        $result = $api->updateUserTunnel($userTunnelId, $tunnelOptions);
        
        flvx_log('ChangePackage - UpdateTunnel', ['userTunnelId' => $userTunnelId, 'options' => $tunnelOptions], $result);
        
        $userResult = $api->updateUser($flvxUserId, [
            'flow' => $configTrafficGb,
            'num' => $configMaxForwards,
            'expTime' => FlvxApiClient::nowMs() + FlvxApiClient::daysToMs($configExpiryDays),
        ]);
        
        flvx_log('ChangePackage - UpdateUser', ['userId' => $flvxUserId], $userResult);
        
        if ($result === false) {
            return ['error' => 'Failed to update package: ' . $api->getLastError()];
        }
        
        return 'success';
        
    } catch (Exception $e) {
        flvx_log('ChangePackage', $params, $e->getMessage(), 'error');
        return ['error' => $e->getMessage()];
    }
}

// ==========================================
// Client Area Functions
// ==========================================

function flvx_ClientArea($params)
{
    try {
        $api = flvx_getApiClient($params);
        $serviceId = $params['serviceid'];
        
        $flvxUserId = flvx_getServiceCustomField($serviceId, 'flvx_user_id');
        $username = flvx_getServiceCustomField($serviceId, 'flvx_username');
        $password = flvx_getServiceCustomField($serviceId, 'flvx_password');
        $tunnelId = flvx_getServiceCustomField($serviceId, 'flvx_tunnel_id');
        
        if (empty($flvxUserId)) {
            return [
                'templatefile' => 'overview',
                'vars' => [
                    'status' => 'not_provisioned',
                    'error' => 'Service not yet provisioned',
                ],
            ];
        }
        
        $configTrafficGb = (int)($params['configoption1'] ?? 100);
        $configMaxForwards = (int)($params['configoption2'] ?? 10);
        
        $userTunnels = $api->listUserTunnels($flvxUserId);
        $userTunnelData = null;
        
        if ($userTunnels !== false) {
            foreach ($userTunnels as $ut) {
                if (isset($ut['tunnelId']) && $ut['tunnelId'] == $tunnelId) {
                    $userTunnelData = $ut;
                    break;
                }
            }
        }
        
        $tunnelInfo = null;
        if (!empty($tunnelId)) {
            $tunnelInfo = $api->getTunnel($tunnelId);
        }
        
        $usedTraffic = 0;
        $totalTraffic = FlvxApiClient::gbToBytes($configTrafficGb);
        if ($userTunnelData) {
            $usedTraffic = ($userTunnelData['inFlow'] ?? 0) + ($userTunnelData['outFlow'] ?? 0);
            $totalTraffic = $userTunnelData['flow'] ?? $totalTraffic;
        }
        
        $expiryTimestamp = 0;
        if ($userTunnelData && isset($userTunnelData['expTime'])) {
            $expiryTimestamp = (int)($userTunnelData['expTime'] / 1000);
        }
        
        $panelUrl = $params['serverhostname'] ?? '';
        if (isset($params['serversecure']) && $params['serversecure']) {
            $panelUrl = 'https://' . $panelUrl;
        } else {
            $panelUrl = 'http://' . $panelUrl;
        }
        
        return [
            'templatefile' => 'overview',
            'vars' => [
                'status' => 'active',
                'username' => $username,
                'password' => $password,
                'tunnelId' => $tunnelId,
                'tunnelName' => $tunnelInfo['name'] ?? 'Unknown',
                'maxForwards' => $configMaxForwards,
                'usedTraffic' => FlvxApiClient::formatBytes($usedTraffic),
                'totalTraffic' => FlvxApiClient::formatBytes($totalTraffic),
                'usedTrafficBytes' => $usedTraffic,
                'totalTrafficBytes' => $totalTraffic,
                'trafficPercentage' => $totalTraffic > 0 ? round(($usedTraffic / $totalTraffic) * 100, 1) : 0,
                'expiryDate' => $expiryTimestamp > 0 ? date('Y-m-d H:i:s', $expiryTimestamp) : 'N/A',
                'expiryTimestamp' => $expiryTimestamp,
                'panelUrl' => $panelUrl,
                'userTunnelData' => $userTunnelData,
            ],
        ];
        
    } catch (Exception $e) {
        return [
            'templatefile' => 'overview',
            'vars' => [
                'status' => 'error',
                'error' => $e->getMessage(),
            ],
        ];
    }
}

function flvx_AdminServicesTabFields($params)
{
    $serviceId = $params['serviceid'];
    
    $flvxUserId = flvx_getServiceCustomField($serviceId, 'flvx_user_id');
    $username = flvx_getServiceCustomField($serviceId, 'flvx_username');
    $tunnelId = flvx_getServiceCustomField($serviceId, 'flvx_tunnel_id');
    $userTunnelId = flvx_getServiceCustomField($serviceId, 'flvx_user_tunnel_id');
    
    return [
        'FLVX User ID' => '<input type="text" name="flvx_user_id" value="' . htmlspecialchars($flvxUserId ?? '') . '" class="form-control" readonly>',
        'FLVX Username' => '<input type="text" name="flvx_username" value="' . htmlspecialchars($username ?? '') . '" class="form-control" readonly>',
        'FLVX Tunnel ID' => '<input type="text" name="flvx_tunnel_id" value="' . htmlspecialchars($tunnelId ?? '') . '" class="form-control" readonly>',
        'FLVX User Tunnel ID' => '<input type="text" name="flvx_user_tunnel_id" value="' . htmlspecialchars($userTunnelId ?? '') . '" class="form-control" readonly>',
    ];
}

function flvx_AdminServicesTabFieldsSave($params)
{
    // FLVX fields are read-only in admin, but we can handle updates if needed
    return;
}

function flvx_ServiceSingleSignOn($params)
{
    try {
        $serviceId = $params['serviceid'];
        $panelUrl = $params['serverhostname'] ?? '';
        
        if (isset($params['serversecure']) && $params['serversecure']) {
            $panelUrl = 'https://' . $panelUrl;
        } else {
            $panelUrl = 'http://' . $panelUrl;
        }
        
        return [
            'redirectTo' => $panelUrl,
        ];
        
    } catch (Exception $e) {
        return ['error' => $e->getMessage()];
    }
}

function flvx_UsageUpdate($params)
{
    try {
        $api = flvx_getApiClient($params);
        
        $services = \WHMCS\Service\Service::where('server', $params['serverid'])
            ->where('domainstatus', 'Active')
            ->get();
        
        foreach ($services as $service) {
            $flvxUserId = flvx_getServiceCustomField($service->id, 'flvx_user_id');
            $userTunnelId = flvx_getServiceCustomField($service->id, 'flvx_user_tunnel_id');
            
            if (empty($flvxUserId) || empty($userTunnelId)) {
                continue;
            }
            
            $userTunnels = $api->listUserTunnels($flvxUserId);
            
            if ($userTunnels === false) {
                continue;
            }
            
            foreach ($userTunnels as $ut) {
                if (isset($ut['id']) && $ut['id'] == $userTunnelId) {
                    $usedBytes = ($ut['inFlow'] ?? 0) + ($ut['outFlow'] ?? 0);
                    $totalBytes = $ut['flow'] ?? 0;
                    
                    if ($totalBytes > 0 && $usedBytes >= $totalBytes) {
                        $service->domainstatus = 'Suspended';
                        $service->save();
                    }
                    
                    break;
                }
            }
        }
        
        return 'success';
        
    } catch (Exception $e) {
        flvx_log('UsageUpdate', $params, $e->getMessage(), 'error');
        return ['error' => $e->getMessage()];
    }
}
