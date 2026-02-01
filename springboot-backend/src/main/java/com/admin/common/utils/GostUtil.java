package com.admin.common.utils;

import cn.hutool.core.util.StrUtil;
import com.admin.common.dto.GostDto;
import com.admin.entity.*;
import com.alibaba.fastjson.JSONArray;
import com.alibaba.fastjson.JSONObject;
import org.apache.commons.lang3.StringUtils;

import java.util.regex.Pattern;
import java.util.List;
import java.util.Map;
import java.util.Objects;

public class GostUtil {


    public static GostDto AddLimiters(Long node_id, Long name, String speed) {
        JSONObject data = createLimiterData(name, speed);
        GostDto gostDto = WebSocketServer.send_msg(node_id, data, "AddLimiters");
        if (gostDto.getMsg().contains("exists")){
            gostDto.setMsg("OK");
        }
        return gostDto;
    }

    public static GostDto UpdateLimiters(Long node_id, Long name, String speed) {
        JSONObject data = createLimiterData(name, speed);
        JSONObject req = new JSONObject();
        req.put("limiter", name + "");
        req.put("data", data);
        return WebSocketServer.send_msg(node_id, req, "UpdateLimiters");
    }

    public static GostDto DeleteLimiters(Long node_id, Long name) {
        JSONObject req = new JSONObject();
        req.put("limiter", name + "");
        GostDto gostDto = WebSocketServer.send_msg(node_id, req, "DeleteLimiters");
        if (gostDto.getMsg().contains("not found")){
            gostDto.setMsg("OK");
        }
        return gostDto;
    }

    public static GostDto AddChains(Long node_id, List<ChainTunnel> chainTunnels, Map<Long, Node> node_s) {
        JSONArray nodes = new JSONArray();
        Node fromNode = node_s.get(node_id);
        for (ChainTunnel chainTunnel : chainTunnels) {
            JSONObject dialer = new JSONObject();
            dialer.put("type", chainTunnel.getProtocol());

            JSONObject connector = new JSONObject();
            connector.put("type", "relay");

            Node node_info = node_s.get(chainTunnel.getNodeId());
            JSONObject node = new JSONObject();
            node.put("name", "node_" + chainTunnel.getInx());

            String dialHost = (fromNode != null && node_info != null)
                    ? selectDialHost(fromNode, node_info)
                    : (node_info != null ? node_info.getServerIp() : null);
            node.put("addr", processServerAddress(dialHost + ":" + chainTunnel.getPort()));
            node.put("connector", connector);
            node.put("dialer", dialer);



            nodes.add(node);
        }
        JSONObject hop = new JSONObject();
        hop.put("name", "hop_" + chainTunnels.getFirst().getTunnelId());

        // interface设置在转发链
        if (StringUtils.isNotBlank(node_s.get(node_id).getInterfaceName())) {
            hop.put("interface", node_s.get(node_id).getInterfaceName());
        }


        JSONObject selector = new JSONObject();
        selector.put("strategy", chainTunnels.getFirst().getStrategy());
        selector.put("maxFails", 1);
        selector.put("failTimeout", 600000000000L); // 600 秒（纳秒单位）


        hop.put("selector", selector);
        hop.put("nodes", nodes);

        JSONArray hops = new JSONArray();
        hops.add(hop);

        JSONObject data = new JSONObject();
        data.put("name", "chains_" + chainTunnels.getFirst().getTunnelId());
        data.put("hops", hops);

        GostDto gostDto = WebSocketServer.send_msg(node_id, data, "AddChains");
        if (gostDto.getMsg().contains("exists")){
            gostDto.setMsg("OK");
        }
        return gostDto;
    }

    public static GostDto DeleteChains(Long node_id, String name) {
        JSONObject data = new JSONObject();
        data.put("chain", name);
        GostDto gostDto = WebSocketServer.send_msg(node_id, data, "DeleteChains");
        if (gostDto.getMsg().contains("not found")){
            gostDto.setMsg("OK");
        }
        return gostDto;
    }

    public static GostDto AddChainService(Long node_id, ChainTunnel chainTunnel, Map<Long, Node> node_s) {
        JSONArray services = new JSONArray();
        Node node_info = node_s.get(chainTunnel.getNodeId());
        JSONObject service_item = new JSONObject();
        service_item.put("name", chainTunnel.getTunnelId() + "_tls");
        service_item.put("addr", node_info.getTcpListenAddr() + ":" + chainTunnel.getPort());
        
        // 只为出口节点(chainType=3)设置 interface
        if (chainTunnel.getChainType() == 3 && StringUtils.isNotBlank(node_s.get(node_id).getInterfaceName())) {
            JSONObject metadata = new JSONObject();
            metadata.put("interface", node_s.get(node_id).getInterfaceName());
            service_item.put("metadata", metadata);
        }

        JSONObject handler = new JSONObject();
        handler.put("type", "relay");
        if (chainTunnel.getChainType() == 2){
            handler.put("chain","chains_" + chainTunnel.getTunnelId());
        }
        service_item.put("handler", handler);

        JSONObject listener = new JSONObject();
        listener.put("type", chainTunnel.getProtocol());
        service_item.put("listener", listener);

        services.add(service_item);

        GostDto gostDto = WebSocketServer.send_msg(node_id, services, "AddService");
        if (gostDto.getMsg().contains("exists")){
            gostDto.setMsg("OK");
        }
        return gostDto;
    }

    public static GostDto AddAndUpdateService(String name, Integer limiter, Node node, Forward forward, ForwardPort forwardPort, Tunnel tunnel, String meth) {
        JSONArray services = new JSONArray();
        String[] protocols = {"tcp", "udp"};
        for (String protocol : protocols) {
            JSONObject service = new JSONObject();
            service.put("name", name + "_" + protocol);
            if (Objects.equals(protocol, "tcp")){
                service.put("addr", node.getTcpListenAddr() + ":" + forwardPort.getPort());
            }else {
                service.put("addr", node.getUdpListenAddr() + ":" + forwardPort.getPort());
            }

            // 只在端口转发时设置 interface（隧道转发时 interface 在转发链的节点上设置）
            if (tunnel.getType() == 1 && StringUtils.isNotBlank(node.getInterfaceName())) {
                JSONObject metadata = new JSONObject();
                metadata.put("interface", node.getInterfaceName());
                service.put("metadata", metadata);
            }

            // 添加限流器配置
            if (limiter != null) {
                service.put("limiter", limiter.toString());
            }

            // 配置处理器
            JSONObject handler = new JSONObject();
            handler.put("type", protocol);
            if (tunnel.getType() == 2){
                handler.put("chain", "chains_" + forward.getTunnelId());
            }
            service.put("handler", handler);

            // 配置监听器
            JSONObject listener = createListener(protocol);
            service.put("listener", listener);

            JSONObject forwarder = createForwarder(forward.getRemoteAddr(), forward.getStrategy());
            service.put("forwarder", forwarder);

            services.add(service);
        }
        GostDto gostDto = WebSocketServer.send_msg(node.getId(), services, meth);
        if (gostDto.getMsg().contains("exists")){
            gostDto.setMsg("OK");
        }
        return gostDto;
    }

    public static GostDto DeleteService(Long node_id, JSONArray services) {
        JSONObject data = new JSONObject();
        data.put("services", services);
        GostDto gostDto = WebSocketServer.send_msg(node_id, data, "DeleteService");
        if (gostDto.getMsg().contains("not found")){
            gostDto.setMsg("OK");
        }
        return gostDto;
    }

    public static GostDto PauseAndResumeService(Long node_id, String name, String meth) {
        JSONObject data = new JSONObject();
        JSONArray services = new JSONArray();
        services.add(name + "_tcp");
        services.add(name + "_udp");
        data.put("services", services);
        return WebSocketServer.send_msg(node_id, data, meth);
    }


    private static JSONObject createLimiterData(Long name, String speed) {
        JSONObject data = new JSONObject();
        data.put("name", name.toString());
        JSONArray limits = new JSONArray();
        limits.add("$ " + speed + "MB " + speed + "MB");
        data.put("limits", limits);
        return data;
    }

    private static JSONObject createListener(String protocol) {
        JSONObject listener = new JSONObject();
        listener.put("type", protocol);
        if (Objects.equals(protocol, "udp")) {
            JSONObject metadata = new JSONObject();
            metadata.put("keepAlive", true);
            listener.put("metadata", metadata);
        }
        return listener;
    }

    private static JSONObject createForwarder(String remoteAddr, String strategy) {
        JSONObject forwarder = new JSONObject();
        JSONArray nodes = new JSONArray();

        String[] split = remoteAddr.split(",");
        int num = 1;
        for (String addr : split) {
            JSONObject node = new JSONObject();
            node.put("name", "node_" + num);
            node.put("addr", addr);
            nodes.add(node);
            num++;
        }

        if (strategy == null || strategy.isEmpty()) {
            strategy = "fifo";
        }

        forwarder.put("nodes", nodes);

        JSONObject selector = new JSONObject();
        selector.put("strategy", strategy);
        selector.put("maxFails", 1);
        selector.put("failTimeout", "600s");
        forwarder.put("selector", selector);
        return forwarder;
    }

    public static String processServerAddress(String serverAddr) {
        if (StrUtil.isBlank(serverAddr)) {
            return serverAddr;
        }

        // 如果已经被方括号包裹，直接返回
        if (serverAddr.startsWith("[")) {
            return serverAddr;
        }

        // 查找最后一个冒号，分离主机和端口
        int lastColonIndex = serverAddr.lastIndexOf(':');
        if (lastColonIndex == -1) {
            // 没有端口号，直接检查是否需要包裹
            return isIPv6Address(serverAddr) ? "[" + serverAddr + "]" : serverAddr;
        }

        String host = serverAddr.substring(0, lastColonIndex);
        String port = serverAddr.substring(lastColonIndex);

        // 检查主机部分是否为IPv6地址
        if (isIPv6Address(host)) {
            return "[" + host + "]" + port;
        }

        return serverAddr;
    }

    private static boolean isIPv6Address(String address) {
        // IPv6地址包含多个冒号，至少2个
        if (!address.contains(":")) {
            return false;
        }

        // 计算冒号数量，IPv6地址至少有2个冒号
        long colonCount = address.chars().filter(ch -> ch == ':').count();
        return colonCount >= 2;
    }

    /**
     * v4 优先：当两端都有 v4 时选择 v4，否则尝试 v6。
     * 用于节点之间建立链路（A -> B 需要选择 B 的地址族，且 A 需要支持该地址族）。
     */
    public static String selectDialHost(Node fromNode, Node toNode) {
        if (fromNode == null || toNode == null) {
            throw new IllegalArgumentException("node is null");
        }

        boolean fromV4 = supportsV4(fromNode);
        boolean fromV6 = supportsV6(fromNode);
        boolean toV4 = supportsV4(toNode);
        boolean toV6 = supportsV6(toNode);

        if (fromV4 && toV4) {
            return pickToAddressV4(toNode);
        }
        if (fromV6 && toV6) {
            return pickToAddressV6(toNode);
        }

        throw new RuntimeException(
                "节点链路不兼容：" + safeName(fromNode) + "(v4=" + fromV4 + ",v6=" + fromV6 + ") -> "
                        + safeName(toNode) + "(v4=" + toV4 + ",v6=" + toV6 + ")"
        );
    }

    private static String safeName(Node node) {
        if (node.getName() == null || node.getName().isBlank()) {
            return "node_" + node.getId();
        }
        return node.getName();
    }

    private static boolean supportsV4(Node node) {
        if (StrUtil.isNotBlank(node.getServerIpV4())) {
            return true;
        }

        String legacy = node.getServerIp();
        if (StrUtil.isBlank(legacy)) {
            return false;
        }

        legacy = legacy.trim();
        if (looksLikeIpv4(legacy)) {
            return true;
        }
        if (isIPv6Address(legacy)) {
            return false;
        }

        // 域名/其它：无法判断，按双栈处理以保持兼容
        return true;
    }

    private static boolean supportsV6(Node node) {
        if (StrUtil.isNotBlank(node.getServerIpV6())) {
            return true;
        }

        String legacy = node.getServerIp();
        if (StrUtil.isBlank(legacy)) {
            return false;
        }

        legacy = legacy.trim();
        if (isIPv6Address(legacy)) {
            return true;
        }
        if (looksLikeIpv4(legacy)) {
            return false;
        }

        // 域名/其它：无法判断，按双栈处理以保持兼容
        return true;
    }

    private static String pickToAddressV4(Node toNode) {
        if (StrUtil.isNotBlank(toNode.getServerIpV4())) {
            return toNode.getServerIpV4().trim();
        }
        String legacy = toNode.getServerIp();
        return legacy != null ? legacy.trim() : null;
    }

    private static String pickToAddressV6(Node toNode) {
        if (StrUtil.isNotBlank(toNode.getServerIpV6())) {
            return toNode.getServerIpV6().trim();
        }
        String legacy = toNode.getServerIp();
        return legacy != null ? legacy.trim() : null;
    }

    private static boolean looksLikeIpv4(String value) {
        Pattern ipv4 = Pattern.compile("^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$");
        return ipv4.matcher(value).matches();
    }
}
