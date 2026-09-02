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
import { expect, test } from 'vitest'

import { buildCompanySidebarData } from './use-sidebar-data'

const t = (key: string) => key

test('company sidebar only exposes approved root entries', () => {
  const titles = buildCompanySidebarData(t as never).navGroups.flatMap(
    (group) => group.items.map((item) => item.title)
  )

  expect(titles).toEqual([
    'Dashboard',
    'API Keys',
    'Usage Logs',
    'Task Logs',
    'Wallet',
    'Profile',
    'Channels',
    'Users',
    'System Info',
  ])

  expect(titles).not.toContain('Playground')
  expect(titles).not.toContain('Chat')
  expect(titles).not.toContain('Overview')
  expect(titles).not.toContain('Models')
  expect(titles).not.toContain('Redemption Codes')
  expect(titles).not.toContain('Subscriptions')
  expect(titles).not.toContain('Task Plugins')
  expect(titles).not.toContain('System Settings')
})
