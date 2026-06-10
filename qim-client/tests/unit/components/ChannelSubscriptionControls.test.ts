import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ChannelListItem from '../../../src/components/channel/ChannelListItem.vue'
import ChannelCard from '../../../src/components/channel/ChannelCard.vue'
import type { Channel } from '../../../src/types'

const defaultChannel: Channel = {
  id: 'default-channel',
  name: '默认频道',
  description: '系统默认频道',
  avatar: '',
  creator_id: 'system',
  status: 'active',
  publish_permission: 'creator_only',
  comment_permission: 'all_subscribers',
  created_at: 1710000000000,
  is_subscribed: true,
  is_default: true,
  subscriber_count: 10,
}

const mountOptions = {
  props: {
    channel: defaultChannel,
    isSelected: false,
  },
  global: {
    stubs: {
      ChannelAvatar: true,
    },
  },
}

describe('channel subscription controls', () => {
  it('does not emit unsubscribe for default subscribed channels in list and card views', async () => {
    const listItem = mount(ChannelListItem, mountOptions)
    const card = mount(ChannelCard, mountOptions)

    expect(listItem.find('.subscribe-btn.default-subscribed').exists()).toBe(true)
    expect(card.find('.card-subscribe-btn.default-subscribed').exists()).toBe(true)

    await listItem.find('.subscribe-btn.default-subscribed').trigger('click')
    await card.find('.card-subscribe-btn.default-subscribed').trigger('click')

    expect(listItem.emitted('unsubscribe')).toBeUndefined()
    expect(card.emitted('unsubscribe')).toBeUndefined()
  })
})
