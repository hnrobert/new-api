/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import * as React from 'react'
import { useTranslation } from 'react-i18next'
import { MultiSelect, type Option } from '@/components/multi-select'
import { getUser, searchUsers } from '@/features/users/api'
import type { User } from '@/features/users/types'

interface UserMultiSelectProps {
  value: number[]
  onChange: (ids: number[]) => void
  placeholder?: string
  id?: string
  disabled?: boolean
}

function userLabel(u: Pick<User, 'display_name' | 'username' | 'id'>): string {
  const name = u.display_name || u.username || `#${u.id}`
  return `${name} (#${u.id})`
}

/**
 * UserMultiSelect — 按用户授权的多选器。
 *
 * 基于 MultiSelect（chips 风格），选项来自按关键字异步搜索的用户（GET /api/user/search），
 * 同时支持 `allowCreate` 直接输入数字用户 ID。已选但尚未搜索到的用户会自动通过
 * GET /api/user/:id 回填显示名；回填失败时降级显示 `#id`。
 */
export function UserMultiSelect(props: UserMultiSelectProps) {
  const { t } = useTranslation()
  const { value, onChange } = props
  const [keyword, setKeyword] = React.useState('')
  const [searchResults, setSearchResults] = React.useState<User[]>([])
  const [known, setKnown] = React.useState<Map<number, User>>(new Map())

  const selectedKey = value.join(',')

  // Prefetch labels for already-selected users not yet known.
  React.useEffect(() => {
    for (const id of value) {
      if (known.has(id)) continue
      getUser(id)
        .then((res) => {
          if (res?.data) {
            setKnown((prev) => {
              if (prev.has(id)) return prev
              const next = new Map(prev)
              next.set(id, res.data)
              return next
            })
          }
        })
        .catch(() => {})
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedKey])

  // Debounced keyword search.
  React.useEffect(() => {
    const kw = keyword.trim()
    if (!kw) {
      setSearchResults([])
      return
    }
    const handle = setTimeout(() => {
      searchUsers({ keyword: kw, page_size: 20 })
        .then((res) => setSearchResults(res?.data?.items ?? []))
        .catch(() => setSearchResults([]))
    }, 250)
    return () => clearTimeout(handle)
  }, [keyword])

  // Merge any newly seen users (from search) into the known map for labels.
  React.useEffect(() => {
    if (searchResults.length === 0) return
    setKnown((prev) => {
      let changed = false
      const next = new Map(prev)
      for (const u of searchResults) {
        if (!next.has(u.id)) {
          next.set(u.id, u)
          changed = true
        }
      }
      return changed ? next : prev
    })
  }, [searchResults])

  const options: Option[] = React.useMemo(() => {
    const map = new Map<string, Option>()
    for (const u of searchResults) {
      map.set(String(u.id), { value: String(u.id), label: userLabel(u) })
    }
    for (const id of value) {
      const key = String(id)
      if (!map.has(key)) {
        const u = known.get(id)
        map.set(key, { value: key, label: u ? userLabel(u) : `#${id}` })
      }
    }
    return Array.from(map.values())
  }, [searchResults, value, known])

  return (
    <MultiSelect
      options={options}
      selected={value.map(String)}
      onChange={(vals) =>
        onChange(
          vals
            .map((v) => Number(v))
            .filter((n) => Number.isInteger(n) && n > 0)
        )
      }
      onInputValueChange={setKeyword}
      placeholder={props.placeholder ?? t('Search users by name / ID')}
      allowCreate
      createLabel={t('Add user ID "{{value}}"')}
      id={props.id}
      disabled={props.disabled}
    />
  )
}
