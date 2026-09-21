/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { TFunction } from 'i18next'
import { Key, LayoutDashboard, ListTodo, Radio, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { SidebarData } from '@/components/layout/types'
import { ROLE } from '@/lib/roles'

/**
 * Root navigation groups for the application sidebar.
 *
 * These are shown when the URL does not match any nested sidebar view
 * registered in `layout/lib/sidebar-view-registry.ts`.
 */
export function buildCompanySidebarData(t: TFunction): SidebarData {
  return {
    navGroups: [
      {
        id: 'general',
        title: t('General'),
        items: [
          {
            title: t('Model Call Analytics'),
            url: '/dashboard/models',
            icon: LayoutDashboard,
          },
          {
            title: t('User Analytics'),
            url: '/dashboard/users',
            icon: Users,
            requiredRole: ROLE.ADMIN,
          },
          {
            title: t('API Keys'),
            url: '/keys',
            icon: Key,
          },
          {
            title: t('Task Logs'),
            url: '/usage-logs/task',
            icon: ListTodo,
          },
        ],
      },
      {
        id: 'admin',
        title: t('Admin'),
        items: [
          {
            title: t('Channels'),
            url: '/channels',
            icon: Radio,
          },
          {
            title: t('Users'),
            url: '/users',
            icon: Users,
          },
        ],
      },
    ],
  }
}

export function filterCompanySidebarDataForRole(
  navGroups: SidebarData['navGroups'],
  role: number
): SidebarData['navGroups'] {
  if (role === ROLE.ADMIN) {
    return navGroups
      .map((group) => ({
        ...group,
        items: group.items.filter(
          (item) =>
            item.url === '/dashboard/models' ||
            item.url === '/dashboard/users' ||
            item.url === '/users'
        ),
      }))
      .filter((group) => group.items.length > 0)
  }

  return navGroups
    .filter((group) => (group.id === 'admin' ? role >= ROLE.ADMIN : true))
    .map((group) => {
      const items = group.items.filter(
        (item) => item.requiredRole === undefined || role >= item.requiredRole
      )
      return items.length === group.items.length ? group : { ...group, items }
    })
}

export function useSidebarData(): SidebarData {
  const { t } = useTranslation()

  return buildCompanySidebarData(t)
}
