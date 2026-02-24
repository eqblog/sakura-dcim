import { create } from 'zustand';
import client from '../api/client';
import type { APIResponse, Tenant } from '../types';

export interface BrandingInfo {
  id?: string;
  name: string;
  slug?: string;
  logo_url?: string;
  primary_color?: string;
  favicon_url?: string;
}

interface BrandingState {
  branding: BrandingInfo;
  loaded: boolean;
  fetchBranding: () => Promise<void>;
  setBrandingFromTenant: (tenant: Tenant) => void;
}

const defaultBranding: BrandingInfo = {
  name: 'Sakura DCIM',
  primary_color: '#667eea',
};

export const useBrandingStore = create<BrandingState>((set) => ({
  branding: defaultBranding,
  loaded: false,

  fetchBranding: async () => {
    try {
      const { data: resp } = await client.get<APIResponse<BrandingInfo>>('/auth/branding');
      if (resp.success && resp.data) {
        set({ branding: { ...defaultBranding, ...resp.data }, loaded: true });
        if (resp.data.favicon_url) {
          const link = document.querySelector<HTMLLinkElement>("link[rel~='icon']");
          if (link) link.href = resp.data.favicon_url;
        }
        if (resp.data.name) {
          document.title = resp.data.name;
        }
      }
    } catch {
      set({ loaded: true });
    }
  },

  setBrandingFromTenant: (tenant: Tenant) => {
    const b: BrandingInfo = {
      id: tenant.id,
      name: tenant.name,
      slug: tenant.slug,
      logo_url: tenant.logo_url,
      primary_color: tenant.primary_color || defaultBranding.primary_color,
      favicon_url: tenant.favicon_url,
    };
    set({ branding: b });
    if (b.favicon_url) {
      const link = document.querySelector<HTMLLinkElement>("link[rel~='icon']");
      if (link) link.href = b.favicon_url;
    }
    if (b.name) document.title = b.name;
  },
}));
