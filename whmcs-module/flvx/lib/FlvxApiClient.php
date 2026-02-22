<?php
/**
 * FLVX API Client for WHMCS Provisioning Module
 * 
 * This class handles all communication with the FLVX panel API.
 * 
 * @package   FLVX WHMCS Module
 * @author    FLVX Team
 * @version   1.0.0
 */

namespace FLVX;

class FlvxApiClient
{
    /**
     * FLVX panel API URL
     * @var string
     */
    private $apiUrl;

    /**
     * JWT authentication token
     * @var string|null
     */
    private $token = null;

    /**
     * Admin username for authentication
     * @var string
     */
    private $adminUsername;

    /**
     * Admin password for authentication
     * @var string
     */
    private $adminPassword;

    /**
     * Token expiry time (Unix timestamp)
     * @var int
     */
    private $tokenExpiry = 0;

    /**
     * Request timeout in seconds
     * @var int
     */
    private $timeout = 30;

    /**
     * Last error message
     * @var string|null
     */
    private $lastError = null;

    /**
     * Constructor
     *
     * @param string $apiUrl FLVX panel URL (e.g., https://panel.example.com)
     * @param string $username Admin username
     * @param string $password Admin password
     */
    public function __construct($apiUrl, $username, $password)
    {
        $this->apiUrl = rtrim($apiUrl, '/');
        $this->adminUsername = $username;
        $this->adminPassword = $password;
    }

    /**
     * Get the last error message
     *
     * @return string|null
     */
    public function getLastError()
    {
        return $this->lastError;
    }

    /**
     * Set the last error message
     *
     * @param string $message
     */
    private function setError($message)
    {
        $this->lastError = $message;
    }

    /**
     * Authenticate and obtain JWT token
     *
     * @return bool True on success, false on failure
     */
    public function authenticate()
    {
        $response = $this->request('/api/v1/user/login', [
            'username' => $this->adminUsername,
            'password' => $this->adminPassword
        ], 'POST', false);

        if ($response === false) {
            return false;
        }

        if (!isset($response['token'])) {
            $this->setError('Authentication failed: No token received');
            return false;
        }

        $this->token = $response['token'];
        // Token is valid for 90 days, refresh 1 day before expiry
        $this->tokenExpiry = time() + (89 * 24 * 60 * 60);

        return true;
    }

    /**
     * Check if current token is valid and refresh if needed
     *
     * @return bool
     */
    private function ensureAuthenticated()
    {
        if ($this->token && time() < $this->tokenExpiry) {
            return true;
        }
        return $this->authenticate();
    }

    /**
     * Make an API request
     *
     * @param string $endpoint API endpoint (e.g., /api/v1/user/create)
     * @param array $data Request data
     * @param string $method HTTP method (GET, POST)
     * @param bool $authRequired Whether authentication is required
     * @return array|false Response data or false on failure
     */
    private function request($endpoint, $data = [], $method = 'POST', $authRequired = true)
    {
        $this->lastError = null;

        $url = $this->apiUrl . $endpoint;
        $ch = curl_init();

        $headers = [
            'Content-Type: application/json',
            'Accept: application/json'
        ];

        // Add authorization header if authenticated and required
        if ($authRequired) {
            if (!$this->ensureAuthenticated()) {
                return false;
            }
            $headers[] = 'Authorization: ' . $this->token;
        }

        curl_setopt_array($ch, [
            CURLOPT_URL => $url,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => $this->timeout,
            CURLOPT_HTTPHEADER => $headers,
            CURLOPT_SSL_VERIFYPEER => true,
            CURLOPT_SSL_VERIFYHOST => 2,
        ]);

        if ($method === 'POST') {
            curl_setopt($ch, CURLOPT_POST, true);
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
        }

        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $curlError = curl_error($ch);
        curl_close($ch);

        if ($curlError) {
            $this->setError('cURL Error: ' . $curlError);
            return false;
        }

        if ($httpCode < 200 || $httpCode >= 300) {
            $this->setError('HTTP Error: ' . $httpCode);
            return false;
        }

        $decoded = json_decode($response, true);
        if (json_last_error() !== JSON_ERROR_NONE) {
            $this->setError('JSON Decode Error: ' . json_last_error_msg());
            return false;
        }

        // Check API response code
        if (isset($decoded['code']) && $decoded['code'] !== 0) {
            $this->setError('API Error: ' . ($decoded['msg'] ?? 'Unknown error'));
            return false;
        }

        return $decoded['data'] ?? $decoded;
    }

    // ==========================================
    // User Management APIs
    // ==========================================

    /**
     * Create a new user
     *
     * @param string $username Username
     * @param string $password Password
     * @param array $options Additional options (flow, num, expTime, status, groupIds)
     * @return array|false User data or false on failure
     */
    public function createUser($username, $password, $options = [])
    {
        $data = array_merge([
            'user' => $username,
            'pwd' => $password,
            'status' => $options['status'] ?? 1,
            'flow' => $options['flow'] ?? 100, // GB
            'num' => $options['num'] ?? 10,
            'expTime' => $options['expTime'] ?? (time() + 365 * 24 * 60 * 60) * 1000, // 1 year in ms
            'flowResetTime' => $options['flowResetTime'] ?? 1,
        ], $options);

        if (isset($options['groupIds'])) {
            $data['groupIds'] = $options['groupIds'];
        }

        return $this->request('/api/v1/user/create', $data);
    }

    /**
     * Update user
     *
     * @param int $userId User ID
     * @param array $data Update data
     * @return array|false
     */
    public function updateUser($userId, $data)
    {
        $data['id'] = $userId;
        return $this->request('/api/v1/user/update', $data);
    }

    /**
     * Delete user
     *
     * @param int $userId User ID
     * @return array|false
     */
    public function deleteUser($userId)
    {
        return $this->request('/api/v1/user/delete', ['id' => $userId]);
    }

    /**
     * Get user by username
     *
     * @param string $username Username to search for
     * @return array|false User data or false if not found
     */
    public function getUserByUsername($username)
    {
        $users = $this->request('/api/v1/user/list', ['keyword' => $username]);
        
        if ($users === false) {
            return false;
        }

        foreach ($users as $user) {
            if (isset($user['user']) && $user['user'] === $username) {
                return $user;
            }
        }

        $this->setError('User not found: ' . $username);
        return false;
    }

    /**
     * Get user package info (tunnels, forwards, traffic stats)
     *
     * Note: This endpoint requires the user's own token, not admin token.
     * For admin access, use getUserByUsername or list users.
     *
     * @return array|false
     */
    public function getUserPackage()
    {
        return $this->request('/api/v1/user/package', []);
    }

    /**
     * Reset user traffic flow
     *
     * @param int $userId User ID
     * @param int $type 1 = reset user flow, 2 = reset user tunnel flow
     * @return array|false
     */
    public function resetUserFlow($userId, $type = 1)
    {
        return $this->request('/api/v1/user/reset', [
            'id' => $userId,
            'type' => $type
        ]);
    }

    // ==========================================
    // Tunnel Management APIs
    // ==========================================

    /**
     * List all tunnels
     *
     * @return array|false
     */
    public function listTunnels()
    {
        return $this->request('/api/v1/tunnel/list', []);
    }

    /**
     * Get tunnel by ID
     *
     * @param int $tunnelId Tunnel ID
     * @return array|false
     */
    public function getTunnel($tunnelId)
    {
        return $this->request('/api/v1/tunnel/get', ['id' => $tunnelId]);
    }

    // ==========================================
    // User-Tunnel Assignment APIs
    // ==========================================

    /**
     * Assign tunnel to user
     *
     * @param int $userId User ID
     * @param int $tunnelId Tunnel ID
     * @param array $options Assignment options
     * @return array|false
     */
    public function assignTunnelToUser($userId, $tunnelId, $options = [])
    {
        $data = [
            'userId' => $userId,
            'tunnelId' => $tunnelId,
            'flow' => $options['flow'] ?? 0, // 0 = unlimited, or bytes
            'num' => $options['num'] ?? 0, // Max forwards
            'expTime' => $options['expTime'] ?? (time() + 365 * 24 * 60 * 60) * 1000, // ms
            'flowResetTime' => $options['flowResetTime'] ?? 1,
            'status' => $options['status'] ?? 1,
        ];

        if (isset($options['speedId'])) {
            $data['speedId'] = $options['speedId'];
        }

        return $this->request('/api/v1/tunnel/user/assign', $data);
    }

    /**
     * Batch assign tunnels to user
     *
     * @param int $userId User ID
     * @param array $tunnels Array of ['tunnelId' => x, 'speedId' => y]
     * @return array|false
     */
    public function batchAssignTunnels($userId, $tunnels)
    {
        return $this->request('/api/v1/tunnel/user/batch-assign', [
            'userId' => $userId,
            'tunnels' => $tunnels
        ]);
    }

    /**
     * Update user-tunnel binding
     *
     * @param int $userTunnelId UserTunnel record ID (NOT user ID or tunnel ID)
     * @param array $options Update options
     * @return array|false
     */
    public function updateUserTunnel($userTunnelId, $options)
    {
        $data = array_merge(['id' => $userTunnelId], $options);
        return $this->request('/api/v1/tunnel/user/update', $data);
    }

    /**
     * Remove user-tunnel binding
     *
     * @param int $userTunnelId UserTunnel record ID
     * @return array|false
     */
    public function removeUserTunnel($userTunnelId)
    {
        return $this->request('/api/v1/tunnel/user/remove', ['id' => $userTunnelId]);
    }

    /**
     * List user's tunnel assignments
     *
     * @param int $userId User ID
     * @return array|false
     */
    public function listUserTunnels($userId)
    {
        return $this->request('/api/v1/tunnel/user/list', ['userId' => $userId]);
    }

    // ==========================================
    // Forward (Port) Management APIs
    // ==========================================

    /**
     * Create a forward (port allocation)
     *
     * @param int $tunnelId Tunnel ID
     * @param string $name Forward name
     * @param string $remoteAddr Remote address (e.g., "8.8.8.8:53")
     * @param array $options Additional options
     * @return array|false
     */
    public function createForward($tunnelId, $name, $remoteAddr, $options = [])
    {
        $data = [
            'tunnelId' => $tunnelId,
            'name' => $name,
            'remoteAddr' => $remoteAddr,
            'strategy' => $options['strategy'] ?? 'fifo',
        ];

        if (isset($options['inPort'])) {
            $data['inPort'] = $options['inPort'];
        }

        return $this->request('/api/v1/forward/create', $data);
    }

    /**
     * List forwards
     *
     * @return array|false
     */
    public function listForwards()
    {
        return $this->request('/api/v1/forward/list', []);
    }

    /**
     * Delete forward
     *
     * @param int $forwardId Forward ID
     * @return array|false
     */
    public function deleteForward($forwardId)
    {
        return $this->request('/api/v1/forward/delete', ['id' => $forwardId]);
    }

    /**
     * Pause forward
     *
     * @param int $forwardId Forward ID
     * @return array|false
     */
    public function pauseForward($forwardId)
    {
        return $this->request('/api/v1/forward/pause', ['id' => $forwardId]);
    }

    /**
     * Resume forward
     *
     * @param int $forwardId Forward ID
     * @return array|false
     */
    public function resumeForward($forwardId)
    {
        return $this->request('/api/v1/forward/resume', ['id' => $forwardId]);
    }

    // ==========================================
    // Speed Limit APIs
    // ==========================================

    /**
     * List speed limits
     *
     * @return array|false
     */
    public function listSpeedLimits()
    {
        return $this->request('/api/v1/speed-limit/list', []);
    }

    // ==========================================
    // Node Management APIs
    // ==========================================

    /**
     * List nodes
     *
     * @return array|false
     */
    public function listNodes()
    {
        return $this->request('/api/v1/node/list', []);
    }

    /**
     * Get node by ID
     *
     * @param int $nodeId Node ID
     * @return array|false
     */
    public function getNode($nodeId)
    {
        $nodes = $this->listNodes();
        if ($nodes === false) {
            return false;
        }

        foreach ($nodes as $node) {
            if (isset($node['id']) && $node['id'] == $nodeId) {
                return $node;
            }
        }

        $this->setError('Node not found: ' . $nodeId);
        return false;
    }

    // ==========================================
    // Utility Methods
    // ==========================================

    /**
     * Convert GB to bytes
     *
     * @param int $gb Gigabytes
     * @return int Bytes
     */
    public static function gbToBytes($gb)
    {
        return $gb * 1024 * 1024 * 1024;
    }

    /**
     * Convert bytes to GB
     *
     * @param int $bytes Bytes
     * @return float Gigabytes
     */
    public static function bytesToGb($bytes)
    {
        return $bytes / (1024 * 1024 * 1024);
    }

    /**
     * Convert days to milliseconds
     *
     * @param int $days Days
     * @return int Milliseconds
     */
    public static function daysToMs($days)
    {
        return $days * 24 * 60 * 60 * 1000;
    }

    /**
     * Get current time in milliseconds
     *
     * @return int
     */
    public static function nowMs()
    {
        return time() * 1000;
    }

    /**
     * Format bytes to human readable
     *
     * @param int $bytes
     * @return string
     */
    public static function formatBytes($bytes)
    {
        $units = ['B', 'KB', 'MB', 'GB', 'TB'];
        $bytes = max($bytes, 0);
        $pow = floor(($bytes ? log($bytes) : 0) / log(1024));
        $pow = min($pow, count($units) - 1);
        $bytes /= pow(1024, $pow);
        return round($bytes, 2) . ' ' . $units[$pow];
    }

    /**
     * Generate a random password
     *
     * @param int $length Password length
     * @return string
     */
    public static function generatePassword($length = 12)
    {
        $chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
        return substr(str_shuffle($chars), 0, $length);
    }
}
