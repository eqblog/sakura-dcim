import React, { useEffect, useState, useCallback } from 'react';
import { message } from 'antd';
import { switchAPI, agentAPI, bandwidthAPI } from '../../api';
import type { Switch, SwitchPort, Agent, VLANSummary, SwitchBandwidthMap } from '../../types';
import SwitchList from './SwitchList';
import SwitchDetail from './SwitchDetail';

const SwitchesPage: React.FC = () => {
  const [switches, setSwitches] = useState<Switch[]>([]);
  const [loading, setLoading] = useState(false);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedSwitch, setSelectedSwitch] = useState<Switch | null>(null);
  const [ports, setPorts] = useState<SwitchPort[]>([]);
  const [portsLoading, setPortsLoading] = useState(false);
  const [vlans, setVlans] = useState<VLANSummary[]>([]);
  const [bandwidth, setBandwidth] = useState<SwitchBandwidthMap>({});

  const fetchSwitches = useCallback(async () => {
    setLoading(true);
    try {
      const { data: resp } = await switchAPI.list();
      if (resp.success) setSwitches(resp.data || []);
    } catch { /* */ }
    setLoading(false);
  }, []);

  useEffect(() => {
    fetchSwitches();
    agentAPI.list({ page: 1, page_size: 200 }).then(({ data: resp }) => {
      if (resp.success) setAgents(resp.data?.items || []);
    });
  }, [fetchSwitches]);

  const fetchSwitchData = useCallback(async (switchId: string) => {
    const [portResp, vlanResp, bwResp] = await Promise.all([
      switchAPI.listPorts(switchId),
      switchAPI.getVLANs(switchId),
      bandwidthAPI.getSwitchBandwidth(switchId),
    ]);
    if (portResp.data.success) setPorts(portResp.data.data || []);
    if (vlanResp.data.success) setVlans(vlanResp.data.data || []);
    if (bwResp.data.success) setBandwidth(bwResp.data.data || {});
  }, []);

  const handleSelectSwitch = useCallback(async (sw: Switch) => {
    setSelectedSwitch(sw);
    setPorts([]);
    setVlans([]);
    setBandwidth({});
    setPortsLoading(true);
    try {
      await fetchSwitchData(sw.id);
    } catch { /* */ }
    setPortsLoading(false);
  }, [fetchSwitchData]);

  const refreshPorts = useCallback(async () => {
    if (!selectedSwitch) return;
    setPortsLoading(true);
    try {
      await fetchSwitchData(selectedSwitch.id);
    } catch { /* */ }
    setPortsLoading(false);
  }, [selectedSwitch, fetchSwitchData]);

  const syncFromSNMP = useCallback(async () => {
    if (!selectedSwitch) return;
    setPortsLoading(true);
    try {
      const { data: resp } = await switchAPI.syncPorts(selectedSwitch.id);
      if (resp.success) {
        message.success('Ports synced from SNMP');
        await fetchSwitchData(selectedSwitch.id);
      } else {
        message.error(resp.error || 'SNMP sync failed');
      }
    } catch {
      message.error('SNMP sync failed');
    }
    setPortsLoading(false);
  }, [selectedSwitch, fetchSwitchData]);

  if (selectedSwitch) {
    return (
      <SwitchDetail
        sw={selectedSwitch}
        ports={ports}
        portsLoading={portsLoading}
        vlans={vlans}
        bandwidth={bandwidth}
        onBack={() => setSelectedSwitch(null)}
        onRefresh={refreshPorts}
        onSyncSNMP={syncFromSNMP}
      />
    );
  }

  return (
    <SwitchList
      switches={switches}
      loading={loading}
      agents={agents}
      onSelect={handleSelectSwitch}
      onRefresh={fetchSwitches}
    />
  );
};

export default SwitchesPage;
