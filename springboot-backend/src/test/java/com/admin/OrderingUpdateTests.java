package com.admin;

import com.admin.common.lang.R;
import com.admin.entity.Node;
import com.admin.entity.Tunnel;
import com.admin.service.NodeService;
import com.admin.service.TunnelService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.jdbc.core.JdbcTemplate;

import javax.annotation.Resource;
import java.math.BigDecimal;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

@SpringBootTest(properties = {
        // use a local sqlite file for tests (resolved via ${DB_PATH} placeholder)
        "DB_PATH=./target/test-gost-ordering.db",
})
class OrderingUpdateTests {

    @Resource
    private JdbcTemplate jdbcTemplate;

    @Resource
    private NodeService nodeService;

    @Resource
    private TunnelService tunnelService;

    @BeforeEach
    void cleanup() {
        // keep it simple; other tables may have foreign references in real runs
        jdbcTemplate.execute("DELETE FROM node");
        jdbcTemplate.execute("DELETE FROM tunnel");
    }

    @Test
    void updateNodeOrder_updatesInx() {
        Node n1 = new Node();
        n1.setName("n1");
        n1.setSecret("s1");
        n1.setServerIp("127.0.0.1");
        n1.setPort("1000-2000");
        n1.setInterfaceName("");
        n1.setHttp(0);
        n1.setTls(0);
        n1.setSocks(0);
        n1.setTcpListenAddr("[::]");
        n1.setUdpListenAddr("[::]");
        n1.setStatus(0);
        n1.setInx(0);
        long now = System.currentTimeMillis();
        n1.setCreatedTime(now);
        n1.setUpdatedTime(now);
        assertTrue(nodeService.save(n1));

        Node n2 = new Node();
        n2.setName("n2");
        n2.setSecret("s2");
        n2.setServerIp("127.0.0.2");
        n2.setPort("1000-2000");
        n2.setInterfaceName("");
        n2.setHttp(0);
        n2.setTls(0);
        n2.setSocks(0);
        n2.setTcpListenAddr("[::]");
        n2.setUdpListenAddr("[::]");
        n2.setStatus(0);
        n2.setInx(0);
        n2.setCreatedTime(now);
        n2.setUpdatedTime(now);
        assertTrue(nodeService.save(n2));

        Node n3 = new Node();
        n3.setName("n3");
        n3.setSecret("s3");
        n3.setServerIp("127.0.0.3");
        n3.setPort("1000-2000");
        n3.setInterfaceName("");
        n3.setHttp(0);
        n3.setTls(0);
        n3.setSocks(0);
        n3.setTcpListenAddr("[::]");
        n3.setUdpListenAddr("[::]");
        n3.setStatus(0);
        n3.setInx(0);
        n3.setCreatedTime(now);
        n3.setUpdatedTime(now);
        assertTrue(nodeService.save(n3));

        List<Map<String, Object>> nodes = new ArrayList<>();
        nodes.add(mapIdInx(n2.getId(), 0));
        nodes.add(mapIdInx(n1.getId(), 1));
        nodes.add(mapIdInx(n3.getId(), 2));

        Map<String, Object> params = new HashMap<>();
        params.put("nodes", nodes);

        R res = nodeService.updateNodeOrder(params);
        assertEquals(0, res.getCode());

        assertEquals(1, nodeService.getById(n1.getId()).getInx());
        assertEquals(0, nodeService.getById(n2.getId()).getInx());
        assertEquals(2, nodeService.getById(n3.getId()).getInx());
    }

    @Test
    void updateTunnelOrder_updatesInx() {
        long now = System.currentTimeMillis();

        Tunnel t1 = new Tunnel();
        t1.setName("t1");
        t1.setType(1);
        t1.setFlow(1);
        t1.setTrafficRatio(new BigDecimal("1.0"));
        t1.setInIp("");
        t1.setStatus(1);
        t1.setInx(0);
        t1.setCreatedTime(now);
        t1.setUpdatedTime(now);
        assertTrue(tunnelService.save(t1));

        Tunnel t2 = new Tunnel();
        t2.setName("t2");
        t2.setType(2);
        t2.setFlow(2);
        t2.setTrafficRatio(new BigDecimal("1.0"));
        t2.setInIp("");
        t2.setStatus(1);
        t2.setInx(0);
        t2.setCreatedTime(now);
        t2.setUpdatedTime(now);
        assertTrue(tunnelService.save(t2));

        List<Map<String, Object>> tunnels = new ArrayList<>();
        tunnels.add(mapIdInx(t2.getId(), 0));
        tunnels.add(mapIdInx(t1.getId(), 1));

        Map<String, Object> params = new HashMap<>();
        params.put("tunnels", tunnels);

        R res = tunnelService.updateTunnelOrder(params);
        assertEquals(0, res.getCode());

        assertEquals(1, tunnelService.getById(t1.getId()).getInx());
        assertEquals(0, tunnelService.getById(t2.getId()).getInx());
    }

    private static Map<String, Object> mapIdInx(Long id, int inx) {
        Map<String, Object> m = new HashMap<>();
        m.put("id", id);
        m.put("inx", inx);
        return m;
    }
}
